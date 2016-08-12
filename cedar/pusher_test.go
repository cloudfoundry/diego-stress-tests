package cedar_test

import (
	"fmt"

	. "code.cloudfoundry.org/diego-stress-tests/cedar"
	"code.cloudfoundry.org/diego-stress-tests/cedar/cedarfakes"
	"golang.org/x/net/context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type FakeCounts struct {
	total        int
	failingPush  int
	failingStart int
}

var _ = Describe("Pusher", func() {
	var config Config
	var pusher Pusher
	var ctx context.Context
	var cancel context.CancelFunc

	generateFakeApps := func(fakeCounts FakeCounts) ([]string, []CfApp) {
		Expect(fakeCounts.total).To(BeNumerically(">", (fakeCounts.failingPush + fakeCounts.failingStart)))

		apps := make([]CfApp, fakeCounts.total)
		appNames := make([]string, fakeCounts.total)

		for i := 0; i < fakeCounts.failingPush; i++ {
			fakeApp := cedarfakes.FakeCfApp{}
			name := fmt.Sprintf("fake-push-failing-app-%d", i)
			fakeApp.AppNameReturns(name)
			fakeApp.PushReturns(fmt.Errorf("failed-to-push"))
			apps[i] = &fakeApp
			appNames[i] = name
		}

		for i := fakeCounts.failingPush; i < (fakeCounts.failingPush + fakeCounts.failingStart); i++ {
			fakeApp := cedarfakes.FakeCfApp{}
			name := fmt.Sprintf("fake-start-failing-app-%d", i)
			fakeApp.AppNameReturns(name)
			fakeApp.GuidReturns(fmt.Sprintf("fake-guid-%d", i), nil)
			fakeApp.StartReturns(fmt.Errorf("failed-to-start"))
			apps[i] = &fakeApp
			appNames[i] = name
		}

		for i := (fakeCounts.failingPush + fakeCounts.failingStart); i < fakeCounts.total; i++ {
			fakeApp := cedarfakes.FakeCfApp{}
			name := fmt.Sprintf("fake-app-%d", i)
			fakeApp.AppNameReturns(name)
			apps[i] = &fakeApp
			appNames[i] = name
		}
		return appNames, apps
	}

	BeforeEach(func() {
		config = Config{
			NumBatches:       1,
			MaxInFlight:      1,
			MaxPollingErrors: 1,
			Tolerance:        0.5,
			Domain:           "bosh-lite.com",
			AppPayload:       "assets/temp-app",
			ConfigFile:       generatedFile,
			OutputFile:       "tmp/output.json",
			Timeout:          30,
		}

		ctx, cancel = context.WithCancel(
			context.WithValue(context.Background(),
				"logger",
				fakeLogger,
			),
		)
	})

	Context("when pushing apps", func() {
		Context("when all apps are pushed succesfully", func() {

			const successfulApps = 3
			var appNames []string
			var apps []CfApp

			BeforeEach(func() {
				appNames, apps = generateFakeApps(FakeCounts{total: successfulApps, failingPush: 0, failingStart: 0})
				pusher = NewPusher(config, apps)
				pusher.PushApps(ctx, cancel)
			})

			It("should have all apps ready to start", func() {
				Expect(len(pusher.AppsToStart)).To(Equal(successfulApps))
				Expect(len(pusher.Report)).To(Equal(successfulApps))
				for _, r := range pusher.Report {
					Expect(appNames).To(ContainElement(*r.AppName))
					Expect(r.PushReport.Succeeded).To(BeTrue())
					Expect(r.PushReport.Duration).NotTo(BeNil())
					Expect(r.PushReport.StartTime).NotTo(BeNil())
					Expect(r.PushReport.EndTime).NotTo(BeNil())
				}
			})
		})

		Context("when some apps fail", func() {

			const successfulApps = 3
			const failedPushes = 1
			var appNames []string
			var apps []CfApp

			BeforeEach(func() {
				appNames, apps = generateFakeApps(FakeCounts{total: successfulApps, failingPush: failedPushes, failingStart: 0})
				pusher = NewPusher(config, apps)
				pusher.PushApps(ctx, cancel)
			})

			FIt("should have some apps ready to start", func() {
				Expect(len(pusher.AppsToStart)).To(Equal(successfulApps - failedPushes))
				Expect(len(pusher.Report)).To(Equal(successfulApps))
			})
		})
	})
})
