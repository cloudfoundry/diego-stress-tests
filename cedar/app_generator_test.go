package main_test

import (
	. "code.cloudfoundry.org/diego-stress-tests/cedar"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
)

var _ = Describe("AppGenerator", func() {
	var config Config
	var ctx context.Context
	var cancel context.CancelFunc
	var cfApps []CfApp

	BeforeEach(func() {
		config = Config{
			NumBatches:       1,
			MaxInFlight:      1,
			MaxPollingErrors: 1,
			Tolerance:        0.5,
			Domain:           "fake-domain.com",
			AppPayload:       "assets/fake-folder",
			ConfigFile:       fakeConfigFile,
			OutputFile:       "tmp/dummy-file.json",
			Timeout:          30,
		}
	})

	JustBeforeEach(func() {
		ctx, cancel = context.WithCancel(
			context.WithValue(context.Background(),
				"logger",
				fakeLogger,
			),
		)

		config.Init(fakeLogger)
		appGenerator := NewAppGenerator(config)
		cfApps = appGenerator.Apps(ctx)
	})

	Context("when a valid config is input", func() {
		It("should generate correct number of cf apps", func() {
			Expect(len(cfApps)).To(Equal(12))
			for _, app := range cfApps {
				Expect(app.AppName()).To(MatchRegexp("light|medium"))
			}
		})
	})
})
