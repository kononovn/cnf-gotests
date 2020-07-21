package sriov

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	_ "github.com/openshift-kni/cnf-gotests/test/network/sriov/tests"
	"github.com/openshift-kni/cnf-gotests/test/util/config"
	testclient "github.com/openshift/sriov-network-operator/test/util/client"
	"github.com/openshift/sriov-network-operator/test/util/k8sreporter"
)

var (
	junitPath  *string
	dumpOutput *bool
)

func init() {
	_, currentFile, _, _ := runtime.Caller(0)
	config := config.NewConfig()
	junitPath = flag.String("junit", fmt.Sprintf("%s", config.GetReportPath(currentFile)), "the path for the junit format report")
	dumpOutput = flag.Bool("dump", false, "dump informations for failed tests")
}

func TestSriov(t *testing.T) {
	RegisterFailHandler(Fail)

	rr := []Reporter{}
	if junitPath != nil {
		rr = append(rr, reporters.NewJUnitReporter(*junitPath))
	}

	reporterFile := os.Getenv("REPORTER_OUTPUT")

	clients := testclient.New("")
	if clients == nil {
		log.Fatal("Client is not set. Please check KUBECONFIG env variable.")
	}

	if reporterFile != "" {
		f, err := os.OpenFile(reporterFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open the file: %v\n", err)
			return
		}
		defer f.Close()
		rr = append(rr, k8sreporter.New(clients, f))

	} else if *dumpOutput {
		rr = append(rr, k8sreporter.New(clients, os.Stdout))
	}

	RunSpecsWithDefaultAndCustomReporters(t, "SRIOV Operator conformance tests", rr)
}
