package seeder_test

import (
	"code.cloudfoundry.org/diego-stress-tests/cedar/config"
	"code.cloudfoundry.org/diego-stress-tests/cedar/config/fakes"
	"code.cloudfoundry.org/diego-stress-tests/cedar/seeder"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppGenerator", func() {
	var cfg *fakes.FakeConfig
	var cfApps []seeder.CfApp

	BeforeEach(func() {
		cfg = &fakes.FakeConfig{}
		cfg.NumBatchesReturns(1)
		cfg.DomainReturns("fake-domain.com")
		cfg.MaxPollingErrorsReturns(1)
		cfg.PrefixReturns("cedarapp")
		cfg.AppTypesReturns([]config.AppDefinition{
			{
				AppNamePrefix: "light",
				AppCount:      9,
			},
			{
				AppNamePrefix: "medium-group",
				AppCount:      3,
			},
		})
	})

	JustBeforeEach(func() {
		appGenerator := seeder.NewAppGenerator(cfg)
		cfApps = appGenerator.Apps(fakeLogger)
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

	Context("when an app prefix is provided", func() {
		BeforeEach(func() {
			cfg.PrefixReturns("cf-2016-08-16T1600")
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
