package seeder_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/diego-stress-tests/cedar/cli"
	"code.cloudfoundry.org/diego-stress-tests/cedar/cli/clifakes"
	"code.cloudfoundry.org/diego-stress-tests/cedar/config"
	"code.cloudfoundry.org/diego-stress-tests/cedar/seeder"
	"code.cloudfoundry.org/diego-stress-tests/cedar/seeder/seederfakes"
	"golang.org/x/net/context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

type FakeCounts struct {
	total, failingPush, failingStart int
	firstAppFailed                   bool
}

const (
	// the value is based on what is set in the config script
	// in seedar_suite_test. do not change unless you update the
	// number of apps in the seeder_suite_test as well
	totalApps = 12
)

var _ = Describe("Deployer", func() {
	var (
		cfg              config.Config
		deployer         seeder.Deployer
		ctx              context.Context
		cancel           context.CancelFunc
		apps             []seeder.CfApp
		appNames         []string
		fakeCli          cli.CFClient
		toleranceNumApps int
	)

	generateFakeApps := func(fakeCounts FakeCounts) ([]string, []seeder.CfApp) {
		Expect(fakeCounts.total).To(BeNumerically(">=", (fakeCounts.failingPush + fakeCounts.failingStart)))

		apps = make([]seeder.CfApp, fakeCounts.total)
		appNames = make([]string, fakeCounts.total)

		for i := 0; i < fakeCounts.failingPush; i++ {
			fakeApp := seederfakes.FakeCfApp{}
			name := fmt.Sprintf("fake-push-failing-app-%d", i)
			fakeApp.AppNameReturns(name)
			fakeApp.PushReturns(fmt.Errorf("failed-to-push"))
			apps[i] = &fakeApp
			appNames[i] = name
		}

		for i := fakeCounts.failingPush; i < (fakeCounts.failingPush + fakeCounts.failingStart); i++ {
			fakeApp := seederfakes.FakeCfApp{}
			name := fmt.Sprintf("fake-start-failing-app-%d", i)
			fakeApp.AppNameReturns(name)
			fakeApp.GuidReturns(fmt.Sprintf("fake-guid-%d", i), nil)
			fakeApp.StartReturns(fmt.Errorf("failed-to-start"))
			apps[i] = &fakeApp
			appNames[i] = name
		}

		for i := (fakeCounts.failingPush + fakeCounts.failingStart); i < fakeCounts.total; i++ {
			fakeApp := seederfakes.FakeCfApp{}
			name := fmt.Sprintf("fake-app-%d", i)
			fakeApp.AppNameReturns(name)
			fakeApp.GuidReturns(fmt.Sprintf("fake-guid-%d", i), nil)
			apps[i] = &fakeApp
			appNames[i] = name
		}

		if !fakeCounts.firstAppFailed {
			apps = append(apps[fakeCounts.total-1:], apps[0:fakeCounts.total-1]...)
			appNames = append(appNames[fakeCounts.total-1:], appNames[0:fakeCounts.total-1]...)
		}

		return appNames, apps
	}

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
		cfg.Init(fakeLogger)

		toleranceNumApps = int(totalApps * cfg.Tolerance)

		fakeCli = &clifakes.FakeCFClient{}

		ctx, cancel = context.WithCancel(
			context.WithValue(context.Background(),
				"logger",
				fakeLogger,
			),
		)
	})

	Context("when pushing apps", func() {
		Context("when all apps are pushed succesfully", func() {
			BeforeEach(func() {
				appNames, apps = generateFakeApps(FakeCounts{total: totalApps, failingPush: 0, failingStart: 0})
				deployer = seeder.NewDeployer(cfg, apps, fakeCli)
				deployer.PushApps(fakeLogger, ctx, cancel)
			})

			It("should have all apps ready to start", func() {
				Expect(len(deployer.AppsToStart)).To(Equal(totalApps))
				Expect(len(deployer.AppStates)).To(Equal(totalApps))
				for _, r := range deployer.AppStates {
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

			JustBeforeEach(func() {
				appNames, apps = generateFakeApps(FakeCounts{total: totalApps, failingPush: failedPushes, failingStart: 0})
				deployer = seeder.NewDeployer(cfg, apps, fakeCli)
				deployer.PushApps(fakeLogger, ctx, cancel)
			})

			Context("when number of failing apps is greater than max failures allowed", func() {
				BeforeEach(func() {
					failedPushes = 9
				})

				It("cancels pushing once tolerance is reached", func() {
					attemptedPushes := len(deployer.AppStates)
					for i := 0; i < toleranceNumApps+1; i++ {
						Expect(fakeLogger).To(gbytes.Say("failed-pushing-app"))
					}
					Expect(fakeLogger).To(gbytes.Say("exceeded-failure-tolerance"))
					for i := 0; i < totalApps-attemptedPushes; i++ {
						Expect(fakeLogger).To(gbytes.Say("push-cancelled"))
					}
					Expect(fakeLogger).To(gbytes.Say("done-pushing-apps"))
				})

				It("records the app state correctly", func() {
					var numFailed = 0
					for _, r := range deployer.AppStates {
						if r.PushState.Succeeded {
							Expect(r.AppGuid).NotTo(BeNil())
							Expect(r.PushState.Duration).NotTo(BeZero())
							Expect(r.PushState.StartTime).NotTo(BeNil())
							Expect(r.PushState.EndTime).NotTo(BeNil())
						} else {
							numFailed++
							Expect(*r.AppGuid).To(BeEquivalentTo(""))
							Expect(r.StartState.Succeeded).To(BeFalse())
							Expect(r.StartState.StartTime).To(BeNil())
							Expect(r.StartState.EndTime).To(BeNil())
							Expect(r.StartState.Duration).To(BeZero())
						}
					}
					Expect(numFailed).To(Equal(toleranceNumApps + 1))
				})
			})

			Context("when number of failing apps is less than max failures allowed", func() {
				BeforeEach(func() {
					failedPushes = 1
				})

				It("pushes all apps except for the failing ones", func() {
					Expect(len(deployer.AppsToStart)).To(Equal(totalApps - failedPushes))
					for i := 0; i < failedPushes; i++ {
						Expect(fakeLogger).To(gbytes.Say("failed-pushing-app"))
					}
					Expect(fakeLogger).To(gbytes.Say("done-pushing-apps"))
				})

				It("records app states", func() {
					Expect(len(deployer.AppStates)).To(Equal(totalApps))
					var numSucceeded = 0
					for _, r := range deployer.AppStates {
						Expect(appNames).To(ContainElement(*r.AppName))
						if r.PushState.Succeeded {
							numSucceeded++
							Expect(r.AppGuid).NotTo(BeNil())
							Expect(r.PushState.Duration).NotTo(BeZero())
							Expect(r.PushState.StartTime).NotTo(BeNil())
							Expect(r.PushState.EndTime).NotTo(BeNil())
						}
					}
					Expect(numSucceeded).To(Equal(len(deployer.AppStates) - failedPushes))
				})
			})
		})

		Context("when the first app fails", func() {
			JustBeforeEach(func() {
				appNames, apps = generateFakeApps(FakeCounts{total: totalApps, failingPush: 1, failingStart: 0, firstAppFailed: true})
				deployer = seeder.NewDeployer(cfg, apps, fakeCli)
				deployer.PushApps(fakeLogger, ctx, cancel)
			})

			It("cancels pushing", func() {
				Expect(len(deployer.AppStates)).To(Equal(1))
			})

			It("records the app state correctly", func() {
				var numFailed = 0
				for _, r := range deployer.AppStates {
					numFailed++
					Expect(*r.AppGuid).To(BeEquivalentTo(""))
					Expect(r.StartState.Succeeded).To(BeFalse())
					Expect(r.StartState.StartTime).To(BeNil())
					Expect(r.StartState.EndTime).To(BeNil())
					Expect(r.StartState.Duration).To(BeZero())
				}
				Expect(numFailed).To(Equal(1))
			})
		})

		Context("when starting apps", func() {
			Context("when all apps are pushed and started succesfully", func() {
				BeforeEach(func() {
					appNames, apps = generateFakeApps(FakeCounts{total: totalApps, failingPush: 0, failingStart: 0})
					deployer = seeder.NewDeployer(cfg, apps, fakeCli)
					deployer.PushApps(fakeLogger, ctx, cancel)
					deployer.StartApps(ctx, cancel)
				})

				It("should have all apps started", func() {
					for _, r := range deployer.AppStates {
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

				JustBeforeEach(func() {
					appNames, apps = generateFakeApps(FakeCounts{total: totalApps, failingPush: 0, failingStart: failedStart})
					deployer = seeder.NewDeployer(cfg, apps, fakeCli)
					deployer.PushApps(fakeLogger, ctx, cancel)
					deployer.StartApps(ctx, cancel)
				})

				Context("when number of apps failing to start is greater than max failures allowed", func() {
					BeforeEach(func() {
						failedStart = 9
					})

					It("cancels starting once tolerance is reached", func() {
						attemptedStart := len(deployer.AppStates)
						for i := 0; i < toleranceNumApps+1; i++ {
							Expect(fakeLogger).To(gbytes.Say("failed-starting-app"))
						}
						Expect(fakeLogger).To(gbytes.Say("failure-tolerance-reached"))
						for i := 0; i < totalApps-attemptedStart; i++ {
							Expect(fakeLogger).To(gbytes.Say("start-cancelled-before-starting-app"))
						}
					})

					It("records the app state correctly", func() {
						var numFailed = 0
						for _, r := range deployer.AppStates {
							if r.StartState.Succeeded {
								Expect(r.StartState.Duration).NotTo(BeZero())
								Expect(r.StartState.StartTime).NotTo(BeNil())
								Expect(r.StartState.EndTime).NotTo(BeNil())
							} else {
								numFailed++
								Expect(r.StartState.Succeeded).To(BeFalse())
							}
						}
						Expect(numFailed).To(BeNumerically(">=", toleranceNumApps+1))
					})
				})

				Context("when number of apps failing to start is less than max failures allowed", func() {
					BeforeEach(func() {
						failedStart = 1
					})

					It("starts all apps except for the failing ones", func() {
						for i := 0; i < len(deployer.AppsToStart)-failedStart; i++ {
							Expect(fakeLogger).To(gbytes.Say("started-app"))
						}
					})

					It("records app states", func() {
						Expect(len(deployer.AppStates)).To(Equal(totalApps))
						var numSucceeded = 0
						for _, r := range deployer.AppStates {
							Expect(appNames).To(ContainElement(*r.AppName))
							if r.StartState.Succeeded {
								numSucceeded++
								Expect(r.StartState.Duration).NotTo(BeZero())
								Expect(r.StartState.StartTime).NotTo(BeNil())
								Expect(r.StartState.EndTime).NotTo(BeNil())
							}
						}
						Expect(numSucceeded).To(Equal(len(deployer.AppStates) - failedStart))
					})
				})
			})
		})

		Context("generate reports", func() {
			var (
				dir, tmpFileName string
				err              error
				jsonParser       *json.Decoder
				failedPushApps   int = 0
			)

			report := struct {
				Succeeded bool                     `json:"succeeded"`
				AppStates []seeder.AppStateMetrics `json:"apps"`
			}{}

			JustBeforeEach(func() {
				dir, err = ioutil.TempDir("", "example")
				Expect(err).NotTo(HaveOccurred())

				tmpFileName = filepath.Join(dir, "tmpfile")
				cfg.OutputFile = tmpFileName

				_, apps = generateFakeApps(FakeCounts{total: totalApps, failingPush: failedPushApps, failingStart: 0})
				deployer = seeder.NewDeployer(cfg, apps, fakeCli)
				deployer.PushApps(fakeLogger, ctx, cancel)
				deployer.StartApps(ctx, cancel)
				deployer.GenerateReport(ctx, cancel)

				Expect(tmpFileName).Should(BeAnExistingFile())
				outputFile, err := os.Open(tmpFileName)
				Expect(err).NotTo(HaveOccurred())

				jsonParser = json.NewDecoder(outputFile)
			})

			AfterEach(func() {
				defer os.RemoveAll(dir) // clean up
			})

			Context("when all apps are pushed succesfully", func() {
				BeforeEach(func() {
					failedPushApps = 3
				})

				It("should generate report successfully", func() {
					err = jsonParser.Decode(&report)
					Expect(err).NotTo(HaveOccurred())
					Expect(report.Succeeded).To(BeTrue())
					Expect(len(report.AppStates)).To(Equal(totalApps))
				})
			})

			Context("when cedar fails", func() {
				BeforeEach(func() {
					failedPushApps = 7
				})

				JustBeforeEach(func() {
					dir, err = ioutil.TempDir("", "example")
					Expect(err).NotTo(HaveOccurred())

					tmpFileName = filepath.Join(dir, "tmpfile")
					cfg.OutputFile = tmpFileName

					_, apps = generateFakeApps(FakeCounts{total: totalApps, failingPush: failedPushApps, failingStart: 0})
					deployer = seeder.NewDeployer(cfg, apps, fakeCli)
					deployer.PushApps(fakeLogger, ctx, cancel)
					deployer.GenerateReport(ctx, cancel)

					Expect(tmpFileName).Should(BeAnExistingFile())
					outputFile, err := os.Open(tmpFileName)
					Expect(err).NotTo(HaveOccurred())

					jsonParser = json.NewDecoder(outputFile)
				})

				It("should generate report successfully", func() {
					err = jsonParser.Decode(&report)
					Expect(err).NotTo(HaveOccurred())
					Expect(report.Succeeded).To(BeFalse())
				})
			})
		})
	})
})
