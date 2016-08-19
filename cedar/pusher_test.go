package main_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	. "code.cloudfoundry.org/diego-stress-tests/cedar"
	"code.cloudfoundry.org/diego-stress-tests/cedar/cedarfakes"
	"golang.org/x/net/context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
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
		Expect(fakeCounts.total).To(BeNumerically(">=", (fakeCounts.failingPush + fakeCounts.failingStart)))

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
			fakeApp.GuidReturns(fmt.Sprintf("fake-guid-%d", i), nil)
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
			Domain:           "fake-domain.com",
			AppPayload:       "assets/fake-folder",
			Prefix:           "cedarapp",
			ConfigFile:       fakeConfigFile,
			OutputFile:       "tmp/dummy-file.json",
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
		var appNames []string
		var apps []CfApp

		Context("when all apps are pushed succesfully", func() {

			var successfulApps = 3

			BeforeEach(func() {
				appNames, apps = generateFakeApps(FakeCounts{total: successfulApps, failingPush: 0, failingStart: 0})
				pusher = NewPusher(config, apps)
				pusher.PushApps(ctx, cancel)
			})

			It("should have all apps ready to start", func() {
				Expect(len(pusher.AppsToStart)).To(Equal(successfulApps))
				Expect(len(pusher.AppStates)).To(Equal(successfulApps))
				for _, r := range pusher.AppStates {
					Expect(appNames).To(ContainElement(*r.AppName))
					Expect(r.AppGuid).NotTo(BeNil())
					Expect(r.PushState.Succeeded).To(BeTrue())
					Expect(r.PushState.Duration).NotTo(BeNil())
					Expect(r.PushState.StartTime).NotTo(BeNil())
					Expect(r.PushState.EndTime).NotTo(BeNil())
				}
			})
		})

		Context("when some apps fail", func() {
			var failedPushes int

			const totalApps = 12 // Set by config
			const tolerance = 6  // Set by config

			JustBeforeEach(func() {
				config.Init(fakeLogger)
				appNames, apps = generateFakeApps(FakeCounts{total: totalApps, failingPush: failedPushes, failingStart: 0})
				pusher = NewPusher(config, apps)
				pusher.PushApps(ctx, cancel)
			})

			Context("when number of failing apps is greater than max failures allowed", func() {
				BeforeEach(func() {
					failedPushes = 9
				})

				It("cancels pushing once tolerance is reached", func() {
					attemptedPushes := len(pusher.AppStates)
					for i := 0; i < tolerance+1; i++ {
						Expect(fakeLogger).To(gbytes.Say("failed-pushing-app"))
					}
					Expect(fakeLogger).To(gbytes.Say("failure-tolerance-reached"))
					for i := 0; i < totalApps-attemptedPushes; i++ {
						Expect(fakeLogger).To(gbytes.Say("push-cancelled-before-pushing-app"))
					}
					Expect(fakeLogger).To(gbytes.Say("done-pushing-apps"))
				})

				It("records the app state correctly", func() {
					var numFailed = 0
					for _, r := range pusher.AppStates {
						if r.PushState.Succeeded {
							Expect(r.AppGuid).NotTo(BeNil())
							Expect(r.PushState.Duration).NotTo(BeZero())
							Expect(r.PushState.StartTime).NotTo(BeNil())
							Expect(r.PushState.EndTime).NotTo(BeNil())
						} else {
							numFailed++
							Expect(r.AppGuid).To(BeNil())
							Expect(r.StartState.Succeeded).To(BeFalse())
							Expect(r.StartState.StartTime).To(BeNil())
							Expect(r.StartState.EndTime).To(BeNil())
							Expect(r.StartState.Duration).To(BeZero())
						}
					}
					Expect(numFailed).To(Equal(tolerance + 1))
				})
			})

			Context("when number of failing apps is less than max failures allowed", func() {
				BeforeEach(func() {
					failedPushes = 1
				})

				It("pushes all apps except for the failing ones", func() {
					Expect(len(pusher.AppsToStart)).To(Equal(totalApps - failedPushes))
					for i := 0; i < failedPushes; i++ {
						Expect(fakeLogger).To(gbytes.Say("failed-pushing-app"))
					}
					Expect(fakeLogger).To(gbytes.Say("done-pushing-apps"))
				})

				It("records app states", func() {
					Expect(len(pusher.AppStates)).To(Equal(totalApps))
					var numSucceeded = 0
					for _, r := range pusher.AppStates {
						Expect(appNames).To(ContainElement(*r.AppName))
						if r.PushState.Succeeded {
							numSucceeded++
							Expect(r.AppGuid).NotTo(BeNil())
							Expect(r.PushState.Duration).NotTo(BeZero())
							Expect(r.PushState.StartTime).NotTo(BeNil())
							Expect(r.PushState.EndTime).NotTo(BeNil())
						}
					}
					Expect(numSucceeded).To(Equal(len(pusher.AppStates) - failedPushes))
				})
			})

		})

		Context("when starting apps", func() {
			var appNames []string
			var apps []CfApp

			Context("when all apps are pushed and started succesfully", func() {

				var successfulApps = 3

				BeforeEach(func() {
					appNames, apps = generateFakeApps(FakeCounts{total: successfulApps, failingPush: 0, failingStart: 0})
					pusher = NewPusher(config, apps)
					pusher.PushApps(ctx, cancel)
					pusher.StartApps(ctx, cancel)
				})

				It("should have all apps started", func() {
					for _, r := range pusher.AppStates {
						Expect(appNames).To(ContainElement(*r.AppName))
						Expect(r.StartState.Succeeded).To(BeTrue())
						Expect(r.StartState.Duration).NotTo(BeZero())
						Expect(r.StartState.StartTime).NotTo(BeNil())
						Expect(r.StartState.EndTime).NotTo(BeNil())
					}
				})
			})

			Context("when some apps fail to start", func() {
				var failedStart int

				const totalApps = 12 // Set by config
				const tolerance = 6  // Set by config

				JustBeforeEach(func() {
					config.Init(fakeLogger)
					appNames, apps = generateFakeApps(FakeCounts{total: totalApps, failingPush: 0, failingStart: failedStart})
					pusher = NewPusher(config, apps)
					pusher.PushApps(ctx, cancel)
					pusher.StartApps(ctx, cancel)
				})

				Context("when number of apps failing to start is greater than max failures allowed", func() {
					BeforeEach(func() {
						failedStart = 9
					})

					It("cancels starting once tolerance is reached", func() {
						attemptedStart := len(pusher.AppStates)
						for i := 0; i < tolerance+1; i++ {
							Expect(fakeLogger).To(gbytes.Say("failed-starting-app"))
						}
						Expect(fakeLogger).To(gbytes.Say("failure-tolerance-reached"))
						for i := 0; i < totalApps-attemptedStart; i++ {
							Expect(fakeLogger).To(gbytes.Say("start-cancelled-before-starting-app"))
						}
					})

					It("records the app state correctly", func() {
						var numFailed = 0
						for _, r := range pusher.AppStates {
							if r.StartState.Succeeded {
								Expect(r.StartState.Duration).NotTo(BeZero())
								Expect(r.StartState.StartTime).NotTo(BeNil())
								Expect(r.StartState.EndTime).NotTo(BeNil())
							} else {
								numFailed++
								Expect(r.StartState.Succeeded).To(BeFalse())
							}
						}
						Expect(numFailed).To(BeNumerically(">=", tolerance+1))
					})
				})

				Context("when number of apps failing to start is less than max failures allowed", func() {
					BeforeEach(func() {
						failedStart = 1
					})

					It("starts all apps except for the failing ones", func() {
						for i := 0; i < len(pusher.AppsToStart)-failedStart; i++ {
							Expect(fakeLogger).To(gbytes.Say("started-app"))
						}
					})

					It("records app states", func() {
						Expect(len(pusher.AppStates)).To(Equal(totalApps))
						var numSucceeded = 0
						for _, r := range pusher.AppStates {
							Expect(appNames).To(ContainElement(*r.AppName))
							if r.StartState.Succeeded {
								numSucceeded++
								Expect(r.StartState.Duration).NotTo(BeZero())
								Expect(r.StartState.StartTime).NotTo(BeNil())
								Expect(r.StartState.EndTime).NotTo(BeNil())
							}
						}
						Expect(numSucceeded).To(Equal(len(pusher.AppStates) - failedStart))
					})
				})
			})
		})

		Context("generate reports", func() {
			var apps []CfApp
			var dir, tmpFileName string
			var err error
			var jsonParser *json.Decoder

			var successfulApps int = 3
			var failedPushApps int = 0

			report := struct {
				Succeeded bool              `json:"succeeded"`
				AppStates []AppStateMetrics `json:"apps"`
			}{}

			JustBeforeEach(func() {

				dir, err = ioutil.TempDir("", "example")
				Expect(err).NotTo(HaveOccurred())

				tmpFileName = filepath.Join(dir, "tmpfile")
				config.OutputFile = tmpFileName

				_, apps = generateFakeApps(FakeCounts{total: successfulApps, failingPush: failedPushApps, failingStart: 0})
				pusher = NewPusher(config, apps)
				pusher.PushApps(ctx, cancel)
				pusher.StartApps(ctx, cancel)
				pusher.GenerateReport(ctx, cancel)

				Expect(tmpFileName).Should(BeAnExistingFile())
				outputFile, err := os.Open(tmpFileName)
				Expect(err).NotTo(HaveOccurred())

				jsonParser = json.NewDecoder(outputFile)
			})

			AfterEach(func() {
				defer os.RemoveAll(dir) // clean up
			})

			Context("when all apps are pushed succesfully", func() {
				It("should generate report successfully", func() {
					err = jsonParser.Decode(&report)
					Expect(err).NotTo(HaveOccurred())
					Expect(report.Succeeded).To(BeTrue())
					Expect(len(report.AppStates)).To(Equal(successfulApps))
				})
			})

			Context("when cedar fails", func() {

				BeforeEach(func() {
					failedPushApps = 2
				})

				AfterEach(func() {
					defer os.RemoveAll(dir) // clean up
				})

				It("should generate report successfully", func() {
					err = jsonParser.Decode(&report)
					Expect(err).NotTo(HaveOccurred())
					Expect(report.Succeeded).To(BeFalse())
					Expect(len(report.AppStates)).To(Equal(failedPushApps))
				})
			})
		})

	})
})
