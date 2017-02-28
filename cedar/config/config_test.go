package config_test

import (
	"time"

	"code.cloudfoundry.org/diego-stress-tests/cedar/cli/fakes"
	. "code.cloudfoundry.org/diego-stress-tests/cedar/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cedar", func() {
	// sample config json file, read and verify, calculating timeout
	var (
		config                                             Config
		err                                                error
		cfClient                                           *fakes.FakeCFClient
		numBatches, maxInFlight, maxPollingErrors          int
		tolerance                                          float64
		domain, appPayload, prefix, configFile, outputFile string
		timeout                                            time.Duration
		useSSL                                             bool
		skipVerifyCertificate                              bool
	)

	BeforeEach(func() {
		numBatches = 1
		maxInFlight = 1
		maxPollingErrors = 1
		tolerance = 0.5
		domain = "bosh-lite.com"
		useSSL = false
		skipVerifyCertificate = false
		appPayload = "assets/temp-app"
		prefix = "cedarapp"
		configFile = fakeConfigFile
		outputFile = "tmp/output.json"
		timeout = 30 * time.Second
		cfClient = &fakes.FakeCFClient{}
	})

	JustBeforeEach(func() {
		config, err = NewConfig(
			fakeLogger,
			cfClient,
			numBatches, maxInFlight, maxPollingErrors,
			tolerance,
			appPayload, prefix, domain, configFile, outputFile,
			timeout,
			useSSL,
			skipVerifyCertificate,
		)
	})

	Context("when passing in a json config", func() {
		It("uses the timeout argument in seconds", func() {
			Expect(config.Timeout()).To(Equal(30 * time.Second))
		})

		It("sets the app count", func() {
			Expect(config.TotalAppCount()).To(Equal(12))
		})

		It("sets the max failures", func() {
			Expect(config.MaxAllowedFailures()).To(Equal(6))
		})

		It("sets the app types", func() {
			Expect(len(config.AppTypes())).To(Equal(2))
			Expect(config.AppTypes()).To(Equal([]AppDefinition{
				AppDefinition{
					ManifestPath:  "manifest-light.yml",
					AppCount:      9,
					AppNamePrefix: "light",
				},
				AppDefinition{
					ManifestPath:  "manifest-medium-group.yml",
					AppCount:      3,
					AppNamePrefix: "medium-group",
				},
			}))
		})

		Context("if the domain is set", func() {
			It("doesn't get shared domains from the cf client", func() {
				Expect(cfClient.CfCallCount()).To(Equal(0))
			})
		})

		Context("if the domain is not set", func() {
			BeforeEach(func() {
				domain = ""
			})

			It("gets shared domains from the cf client", func() {
				Expect(cfClient.CfCallCount()).To(Equal(1))
				_, _, _, args := cfClient.CfArgsForCall(0)
				Expect(args).To(Equal([]string{"curl", "/v2/shared_domains"}))
			})
		})
	})
})
