package diagnosis_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDiagnosis(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Diagnosis Suite")
}

var _ = BeforeEach(func() {
	// bbsServer = ghttp.NewServer()
})

var _ = AfterEach(func() {
	// bbsServer.CloseClientConnections()
	// bbsServer.Close()
})
