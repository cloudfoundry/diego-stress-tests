package cli_test

import (
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var fakeLogger lager.Logger

func TestCli(t *testing.T) {
	BeforeEach(func() {
		fakeLogger = lagertest.NewTestLogger("fakelogger")
	})
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cli Suite")
}
