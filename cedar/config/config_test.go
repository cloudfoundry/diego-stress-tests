package config_test

import (
	"time"

	. "code.cloudfoundry.org/diego-stress-tests/cedar/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cedar", func() {
	// sample config json file, read and verify, calculating timeout
	var config Config

	BeforeEach(func() {
		config = Config{
			NumBatches:       1,
			MaxInFlight:      1,
			MaxPollingErrors: 1,
			Tolerance:        0.5,
			Domain:           "bosh-lite.com",
			AppPayload:       "assets/temp-app",
			Prefix:           "cedarapp",
			ConfigFile:       fakeConfigFile,
			OutputFile:       "tmp/output.json",
			Timeout:          30,
		}
	})

	Context("when passing in a json config", func() {

		BeforeEach(func() {
			config.Init(fakeLogger)
		})

		It("uses the timeout argument in seconds", func() {
			Expect(config.TimeoutDuration()).To(Equal(30 * time.Second))
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

	})
})
