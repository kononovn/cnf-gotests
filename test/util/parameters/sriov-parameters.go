package parameters

import (
	"fmt"
	"os"
)

const (
	MTUJumbo                          = 9000
	MTUCustom                         = 1450
	MTUStandart                       = 1500
	ConnectivityDiffNode              = "2 pods on different node"
	ConnectivitySameNodeDiffPF        = "2 pods on the same node 2 different PF"
	ConnectivitySameNodeSamePF        = "2 pods on same node same PF"
	ConnectivityPodExtPodInt          = "pod to external/External to pod"
	CommunicationProtocolUnicastICMP  = "unicast-icmp"
	CommunicationProtocolUnicastTCP   = "unicast-tcp"
	CommunicationProtocolUnicastUDP   = "unicast-udp"
	CommunicationProtocolMulticastUDP = "multicast-udp"
	CommunicationProtocolBroadcastUDP = "broadcast-udp"
	CommunicationProtocolSctpUDP      = "unicast-sctp"
)

var mtuParameters = []int{MTUCustom, MTUJumbo, MTUStandart}
var connectivityParameters = []string{
	ConnectivityDiffNode, ConnectivitySameNodeDiffPF, ConnectivitySameNodeSamePF, ConnectivityPodExtPodInt}

var protocolParameters = []string{CommunicationProtocolUnicastICMP, CommunicationProtocolUnicastTCP,
	CommunicationProtocolUnicastUDP, CommunicationProtocolMulticastUDP,
	CommunicationProtocolBroadcastUDP, CommunicationProtocolSctpUDP}

// ConnectivityTestParameters contains test parametrs for connectivity
type ConnectivityTestParameters struct {
	Protocol     string
	MTU          int
	Connectivity string
}

// NewConnectivityTestParameters creates new instance of ConnectivityTestParameters
func NewConnectivityTestParameters(MTU int, Connectivity string, Protocol string) *ConnectivityTestParameters {
	connectivityTestParameters := new(ConnectivityTestParameters)
	validateIntParam(MTU, mtuParameters)
	connectivityTestParameters.MTU = MTU
	validateSrtParam(Connectivity, connectivityParameters)
	connectivityTestParameters.Connectivity = Connectivity
	validateSrtParam(Protocol, protocolParameters)
	connectivityTestParameters.Protocol = Protocol
	return connectivityTestParameters
}

func validateIntParam(intParam int, intParamRange []int) {
	for _, parameter := range intParamRange {
		if intParam == parameter {
			return
		}
	}
	fmt.Fprintf(os.Stderr, "error: %v\n", fmt.Errorf("Wrong Parameter %v", intParam))
	os.Exit(1)
}

func validateSrtParam(param string, paramRange []string) {
	for _, parameter := range paramRange {
		if param == parameter {
			return
		}
	}
	fmt.Fprintf(os.Stderr, "error: %v\n", fmt.Errorf("Wrong Parameter %v", param))
	os.Exit(1)
}
