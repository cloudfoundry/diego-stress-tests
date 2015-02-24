package diego_test

import (
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

const (
	CF_PUSH_TIMEOUT    = 1 * time.Minute
	CF_LOGS_TIMEOUT    = 10 * time.Second
	CF_SET_ENV_TIMEOUT = 10 * time.Second
	CF_START_TIMEOUT   = 4 * time.Minute
	LONG_CURL_TIMEOUT  = 4 * time.Minute

	DIEGO_STAGE_BETA = "DIEGO_STAGE_BETA"
	DIEGO_RUN_BETA   = "DIEGO_RUN_BETA"
)

var context helpers.SuiteContext
var stress_test_data_dir = os.Getenv("STRESS_TEST_DATA_DIR")

func TestApplications(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	config := helpers.LoadConfig()
	context = helpers.NewContext(config)
	environment := helpers.NewEnvironment(context)

	BeforeSuite(func() {
		environment.Setup()
		err := os.RemoveAll(stress_test_data_dir)
		Ω(err).ShouldNot(HaveOccurred())
	})

	componentName := "Diego"

	rs := []Reporter{}

	if config.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(config, componentName)
		rs = append(rs, helpers.NewJUnitReporter(config, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}
