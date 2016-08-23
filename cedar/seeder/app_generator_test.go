package seeder_test

import (
	"code.cloudfoundry.org/diego-stress-tests/cedar/config"
	"code.cloudfoundry.org/diego-stress-tests/cedar/seeder"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
)

var _ = Describe("AppGenerator", func() {
	var cfg config.Config
	var ctx context.Context
	var cfApps []seeder.CfApp

	BeforeEach(func() {
		cfg = config.Config{
			NumBatches:       1,
			MaxInFlight:      1,
			MaxPollingErrors: 1,
			Tolerance:        0.5,
			Domain:           "fake-domain.com",
			AppPayload:       "assets/fake-folder",
			Prefix:           "cedarapp",
			ConfigFile:       fakeConfigFile,
			OutputFile:       "tmp/dummy-file.json",
			Timeout:          30,
		}
	})

	JustBeforeEach(func() {
		ctx, _ = context.WithCancel(
			context.WithValue(context.Background(),
				"logger",
				fakeLogger,
			),
		)

		cfg.Init(fakeLogger)
		appGenerator := seeder.NewAppGenerator(cfg)
		cfApps = appGenerator.Apps(ctx)
	})

	Context("when a valid config is provided", func() {
		It("should generate correct number of cf apps", func() {
			Expect(len(cfApps)).To(Equal(12))
			for _, app := range cfApps {
				Expect(app.AppName()).To(
					MatchRegexp(`cedarapp-\d-light-\d|cedarapp-\d-medium-group-\d`),
				)
			}
		})
	})

	Context("when an app prefix is porvided", func() {
		BeforeEach(func() {
			cfg.Prefix = "cf-2016-08-16T1600"
		})
		It("should generate correct number of cf apps", func() {
			Expect(len(cfApps)).To(Equal(12))
			for _, app := range cfApps {
				Expect(app.AppName()).To(
					MatchRegexp(`cf-2016-08-16T1600-\d-light-\d|cf-2016-08-16T1600-\d-medium-group-\d`),
				)
			}
		})
	})
})
