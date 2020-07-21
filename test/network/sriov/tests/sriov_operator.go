package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/cnf-gotests/test/util/cluster"
	"github.com/openshift-kni/cnf-gotests/test/util/execute"
	"github.com/openshift-kni/cnf-gotests/test/util/namespaces"
	"github.com/openshift-kni/cnf-gotests/test/util/parameters"
	"github.com/openshift-kni/cnf-gotests/test/util/pod"
	sriovv1 "github.com/openshift/sriov-network-operator/pkg/apis/sriovnetwork/v1"
	testclient "github.com/openshift/sriov-network-operator/test/util/client"
	corev1 "k8s.io/api/core/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	namespace = "sriov-operator-tests"
)

var (
	waitingTime time.Duration = 20 * time.Minute
)

var _ = Describe("CNF SRIOV", func() {
	describe := func(desc string) func(*parameters.ConnectivityTestParameters) string {

		return func(connectivityParameters *parameters.ConnectivityTestParameters) string {

			myPrams, err := json.Marshal(connectivityParameters)
			if err != nil {
				fmt.Printf("Error: %s", err)
			}

			return fmt.Sprintf("%s %s", desc, string(myPrams))
		}
	}
	usualSriovPolicyConfig := getSriovPolicy("test-policy-usual", "eno1", "#0-1", "testresourceusual")
	usualSriovNetworkConfig := getSriovNetwork("test-sriov-static-usual", usualSriovPolicyConfig.Spec.ResourceName)
	customSriovPolicyConfig := getSriovPolicy("test-policy-custom", "eno1", "#2-3", "testresourcecustom")
	customSriovNetworkConfig := getSriovNetwork("test-sriov-static-custom", customSriovPolicyConfig.Spec.ResourceName)
	jumboSriovPolicyConfig := getSriovPolicy("test-policy-jumbo", "eno1", "#4-4", "testresourcejumbo")
	jumboSriovNetworkConfig := getSriovNetwork("test-sriov-static-jumbo", jumboSriovPolicyConfig.Spec.ResourceName)
	var sriovInfos *cluster.EnabledNodes
	var err error

	execute.BeforeAll(func() {
		removeAllSriovNetworks()
		removeAllSriovPolicy()
		sriovInfos, err = cluster.DiscoverSriov(clients, operatorNamespace)
		Expect(err).ToNot(HaveOccurred())
		clients := testclient.New("")
		Expect(clients).NotTo(BeNil(), "Client misconfigured, check the $KUBECONFIG env variable")
		waitForSRIOVStable()
		err := namespaces.Create(namespace, clients)
		Expect(err).ToNot(HaveOccurred())
		err = clients.Create(context.Background(), usualSriovPolicyConfig)
		Expect(err).ToNot(HaveOccurred())
		customSriovPolicyConfig.Spec.Mtu = 1450
		err = clients.Create(context.Background(), customSriovPolicyConfig)
		jumboSriovPolicyConfig.Spec.Mtu = 9000
		Expect(err).ToNot(HaveOccurred())
		err = clients.Create(context.Background(), jumboSriovPolicyConfig)
		Expect(err).ToNot(HaveOccurred())
		err = clients.Create(context.Background(), usualSriovNetworkConfig)
		Expect(err).ToNot(HaveOccurred())
		err = clients.Create(context.Background(), customSriovNetworkConfig)
		Expect(err).ToNot(HaveOccurred())
		err = clients.Create(context.Background(), jumboSriovNetworkConfig)
		Expect(err).ToNot(HaveOccurred())
		waitForSRIOVStable()
		//CNF Sriov: Ipam type: IP Static, Ip Stack: Dual-stack, Mac address: MAC static
	})

	// Need to move this function to sriov_suite_test.go
	AfterSuite(func() {
		var timeout time.Duration
		timeout = 1800 * time.Second
		removeAllSriovNetworks()
		removeAllSriovPolicy()
		err = namespaces.DeleteAndWait(clients, namespace, timeout)
		Expect(err).ToNot(HaveOccurred())
		waitForSRIOVStable()
	})

	//First table for positive tests
	DescribeTable("IP Static, Ip Stack: Dual-stack, Mac address: MAC static", func(connectivityParameters *parameters.ConnectivityTestParameters) {
		clinetPodIP := "192.168.100.1"
		serverPodIP := "192.168.100.2"

		// Here we need to generate only arguments for test container which will run relevant command by
		// preconfigured script. Also we need to configure negative flag in order to cover negative scenarious
		var testCommand []string
		if connectivityParameters.Protocol == parameters.CommunicationProtocolUnicastICMP {
			testCommand = []string{"ping", serverPodIP, "-c", "5"}
		}
		var networkName string
		if connectivityParameters.MTU == parameters.MTUJumbo {
			networkName = jumboSriovNetworkConfig.Name
			testCommand = append(testCommand, "-s", "8972", "-M", "do")
		} else if connectivityParameters.MTU == parameters.MTUStandart {
			networkName = usualSriovNetworkConfig.Name
			testCommand = append(testCommand, "-s", "1460", "-M", "do")
		} else if connectivityParameters.MTU == parameters.MTUCustom {
			networkName = customSriovNetworkConfig.Name
			testCommand = append(testCommand, "-s", "1400", "-M", "do")
		} else {
			Skip(fmt.Sprintf("Unsupported test parameter %d", connectivityParameters.MTU))
		}
		fmt.Println(testCommand)
		var nodeSelector []string
		if connectivityParameters.Connectivity == parameters.ConnectivityDiffNode {
			if len(sriovInfos.Nodes) < 2 {
				Skip("Nodes number less that 2")
			}
			nodeSelector = sriovInfos.Nodes
		} else if connectivityParameters.Connectivity == parameters.ConnectivitySameNodeSamePF {
			nodeSelector = append(nodeSelector, sriovInfos.Nodes[0])
		} else {
			Skip(fmt.Sprintf("Unsupported test parameter %s", connectivityParameters.Connectivity))
		}

		var serverPodDefinition *corev1.Pod
		var clientPodDefinition *corev1.Pod
		if len(nodeSelector) > 1 {
			serverPodDefinition = pod.DefineWithNodeNetworks(nodeSelector[0], []string{networkName}, namespace, "centos")
			clientPodDefinition = pod.RedefineWithRestartPolicy(pod.DefineWithNodeNetworks(nodeSelector[1], []string{networkName}, namespace, "centos"),
				corev1.RestartPolicyNever)
		} else {
			serverPodDefinition = pod.DefineWithNodeNetworks(nodeSelector[0], []string{networkName}, namespace, "centos")
			clientPodDefinition = pod.RedefineWithRestartPolicy(pod.DefineWithNodeNetworks(nodeSelector[0], []string{networkName}, namespace, "centos"),
				k8sv1.RestartPolicyNever)
		}
		serverPodDefinition.Annotations = map[string]string{"k8s.v1.cni.cncf.io/networks": fmt.Sprintf(`[
					{
						"name": "%s", 
						"mac": "20:04:0f:f1:88:01",
						"ips": ["%s/24"]
					}
				]`, networkName, serverPodIP)}
		clientPodDefinition.Annotations = map[string]string{"k8s.v1.cni.cncf.io/networks": fmt.Sprintf(`[
					{
						"name": "%s", 
						"mac": "20:04:0f:f1:88:03",
						"ips": ["%s/24"]
					}
				]`, networkName, clinetPodIP)}

		serverPodDefinition.Spec.Containers[0].Command = []string{"sleep", "3600"}
		clientPodDefinition.Spec.Containers[0].Command = testCommand
		serverPod, err := clients.Pods(namespace).Create(context.Background(), serverPodDefinition, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() corev1.PodPhase {
			serverPod, _ = clients.Pods(namespace).Get(context.Background(), serverPod.Name, metav1.GetOptions{})
			return serverPod.Status.Phase
		}, 3*time.Minute, time.Second).Should(Equal(corev1.PodRunning))

		clientPod, err := clients.Pods(namespace).Create(context.Background(), clientPodDefinition, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		defer deletePod(clientPod)
		defer deletePod(serverPod)
		Eventually(func() corev1.PodPhase {
			clientPod, _ = clients.Pods(namespace).Get(context.Background(), clientPod.Name, metav1.GetOptions{})
			return clientPod.Status.Phase
		}, 3*time.Minute, time.Second).Should(Equal(k8sv1.PodSucceeded))
	},

		Entry(describe(""),
			parameters.NewConnectivityTestParameters(parameters.MTUCustom, parameters.ConnectivityDiffNode,
				parameters.CommunicationProtocolUnicastICMP)),
		Entry(describe(""),
			parameters.NewConnectivityTestParameters(parameters.MTUCustom, parameters.ConnectivitySameNodeSamePF,
				parameters.CommunicationProtocolUnicastICMP)),
		Entry(describe(""),
			parameters.NewConnectivityTestParameters(parameters.MTUStandart, parameters.ConnectivityDiffNode,
				parameters.CommunicationProtocolUnicastICMP)),
		Entry(describe(""),
			parameters.NewConnectivityTestParameters(parameters.MTUStandart, parameters.ConnectivitySameNodeSamePF,
				parameters.CommunicationProtocolUnicastICMP)),
		Entry(describe(""),
			parameters.NewConnectivityTestParameters(parameters.MTUJumbo, parameters.ConnectivityDiffNode,
				parameters.CommunicationProtocolUnicastICMP)),
		Entry(describe(""),
			parameters.NewConnectivityTestParameters(parameters.MTUJumbo, parameters.ConnectivitySameNodeSamePF,
				parameters.CommunicationProtocolUnicastICMP)),
	)
})

func deletePod(podObj *k8sv1.Pod) {
	err := clients.Pods(namespace).Delete(context.Background(), podObj.Name, metav1.DeleteOptions{
		GracePeriodSeconds: pointer.Int64Ptr(0)})
	Expect(err).ToNot(HaveOccurred())
}

func getSriovNetwork(name string, resourceName string) *sriovv1.SriovNetwork {
	return &sriovv1.SriovNetwork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: operatorNamespace,
		},
		Spec: sriovv1.SriovNetworkSpec{
			ResourceName:     resourceName,
			IPAM:             `{ "type": "static" }`,
			Capabilities:     `{ "mac": true, "ips": true }`,
			NetworkNamespace: namespace,
		}}
}

func getSriovPolicy(name string, intName string, pfRange string, resourceName string) *sriovv1.SriovNetworkNodePolicy {
	return &sriovv1.SriovNetworkNodePolicy{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name,
			Namespace:    operatorNamespace,
		},

		Spec: sriovv1.SriovNetworkNodePolicySpec{
			NodeSelector: map[string]string{
				"node-role.kubernetes.io/worker-cnf": "",
			},
			NumVfs:       5,
			ResourceName: resourceName,
			Priority:     99,
			NicSelector: sriovv1.SriovNetworkNicSelector{
				PfNames: []string{intName + pfRange},
			},
			DeviceType: "netdevice",
		},
	}
}

func waitForSRIOVStable() {
	// This used to be to check for sriov not to be stable first,
	// then stable. The issue is that if no configuration is applied, then
	// the status won't never go to not stable and the test will fail.
	// TODO: find a better way to handle this scenario

	time.Sleep(5 * time.Second)
	Eventually(func() bool {
		res, err := cluster.SriovStable(operatorNamespace, clients)
		Expect(err).ToNot(HaveOccurred())
		return res
	}, waitingTime, 1*time.Second).Should(BeTrue())

	Eventually(func() bool {
		isClusterReady, err := cluster.IsClusterStable(clients)
		Expect(err).ToNot(HaveOccurred())
		return isClusterReady
	}, waitingTime, 1*time.Second).Should(BeTrue())
}

func removeAllSriovPolicy() {
	sriovPolicies, err := clients.SriovNetworkNodePolicies(operatorNamespace).List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	for _, sriovPolicy := range sriovPolicies.Items {
		if sriovPolicy.Name != "default" {
			err = clients.Delete(context.Background(), &sriovPolicy)
			Expect(err).ToNot(HaveOccurred())
		}
	}
}

func removeAllSriovNetworks() {
	sriovNetworks, err := clients.SriovNetworks(operatorNamespace).List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	for _, sriovNetwork := range sriovNetworks.Items {
		err = clients.Delete(context.Background(), &sriovNetwork)
		Expect(err).ToNot(HaveOccurred())
	}
}
