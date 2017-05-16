package diagnosis_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/diego-stress-tests/drd/diagnosis"
	"code.cloudfoundry.org/diego-stress-tests/drd/parser"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Diagnosis", func() {

	Describe("DiagnoseProcessGuid", func() {
		var (
			app         *parser.App
			desiredLRPs []*models.DesiredLRP
			processGuid string
		)

		instanceInfo := func(app, appGuid, processGuid, instanceGuid, cellId, state string, index int32) diagnosis.InstanceInfo {
			return diagnosis.InstanceInfo{
				InstanceGuid: instanceGuid,
				CellId:       cellId,
				AppName:      app,
				AppGuid:      appGuid,
				ProcessGuid:  processGuid,
				State:        state,
				Index:        index,
			}
		}

		actualLRPInfo := func(processGuid, instanceGuid, state, cellId string, index int32) *models.ActualLRPGroup {
			return &models.ActualLRPGroup{
				Instance: &models.ActualLRP{
					ActualLRPKey: models.ActualLRPKey{
						ProcessGuid: processGuid,
						Index:       index,
						Domain:      "domain",
					},
					ActualLRPInstanceKey: models.ActualLRPInstanceKey{
						InstanceGuid: instanceGuid,
						CellId:       cellId,
					},
					State: state,
				},
			}
		}

		BeforeEach(func() {
			processGuid = "app-guid-process-guid"
			app = &parser.App{
				Name: "app",
				Guid: "app-guid",
			}

			desiredLRPs = []*models.DesiredLRP{
				&models.DesiredLRP{
					ProcessGuid: processGuid,
					Instances:   1,
				},
				&models.DesiredLRP{
					ProcessGuid: "not-app-guid-another-process-guid",
					Instances:   1,
				},
			}
		})

		Context("DiscoverProcessGuid", func() {
			It("returns the process guid of the app", func() {
				Expect(diagnosis.DiscoverProcessGuid(app, desiredLRPs)).To(Equal(desiredLRPs[0]))
			})
		})

		Context("DiscoverActualLRPs", func() {
			var (
				expectedActualLRPGroups []*models.ActualLRPGroup
				cellId                  string
				instance0               string
				instance1               string

				runningInstance   diagnosis.InstanceInfo
				unclaimedInstance diagnosis.InstanceInfo
				crashedInstance   diagnosis.InstanceInfo
				claimedInstance   diagnosis.InstanceInfo

				appSummary diagnosis.Summary
			)

			BeforeEach(func() {
				cellId = "cell-id"
				instance0 = "instance-guid-0"
				instance1 = "instance-guid-1"

				expectedActualLRPGroups = []*models.ActualLRPGroup{
					actualLRPInfo(processGuid, instance0, "RUNNING", cellId, 0),
					actualLRPInfo(processGuid, instance1, "CLAIMED", cellId, 1),
					actualLRPInfo(processGuid, "", "UNCLAIMED", "", 2),
					actualLRPInfo(processGuid, "", "CRASHED", "", 3),
				}

				runningInstance = instanceInfo(app.Name, app.Guid, processGuid, instance0, cellId, "RUNNING", 0)
				claimedInstance = instanceInfo(app.Name, app.Guid, processGuid, instance1, cellId, "CLAIMED", 1)

				unclaimedInstance = instanceInfo(app.Name, app.Guid, processGuid, "", "", "UNCLAIMED", 2)
				crashedInstance = instanceInfo(app.Name, app.Guid, processGuid, "", "", "CRASHED", 3)

				appSummary = diagnosis.DiagnoseApp(app, *desiredLRPs[0], expectedActualLRPGroups)
			})

			It("collects app summary based on desired and actual lrp", func() {
				Expect(appSummary.InstanceSummary.Tracked.Running).To(Equal(1))
				Expect(appSummary.InstanceSummary.Tracked.Claimed).To(Equal(1))
				Expect(appSummary.InstanceSummary.Untracked.Unclaimed).To(Equal(1))
				Expect(appSummary.InstanceSummary.Untracked.Crashed).To(Equal(1))

				Expect(appSummary.TrackedInstances).To(ContainElement(runningInstance))
				Expect(appSummary.TrackedInstances).To(ContainElement(claimedInstance))
				Expect(appSummary.UntrackedInstances).To(ContainElement(unclaimedInstance))
				Expect(appSummary.UntrackedInstances).To(ContainElement(crashedInstance))
			})

			Context("when writing to a file", func() {
				var (
					filePath, dir  string
					expectedOutput string
					err            error
				)

				BeforeEach(func() {
					dir, err = ioutil.TempDir("/tmp", "report")
					Expect(err).NotTo(HaveOccurred())

					filePath = fmt.Sprintf("%s/summary.json", dir)

					content, err := ioutil.ReadFile("fixtures/output.json")
					expectedOutput = strings.Replace(string(content), " ", "", -1)
					expectedOutput = strings.Replace(expectedOutput, "\n", "", -1)
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func() {
					os.RemoveAll(dir)
				})

				It("generates a report", func() {
					layout := "2006-01-02T15:04:05.999999999-07:00"
					timeVal, err := time.Parse(layout, "2017-05-15T15:23:57.367072665-07:00")
					appSummary.Timestamp = timeVal
					Expect(err).NotTo(HaveOccurred())
					diagnosis.WriteToFile(appSummary, filePath)
					content, err := ioutil.ReadFile(filePath)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(content)).To(Equal(expectedOutput))
				})
			})
		})

		Context("JoinSummaries", func() {
			var (
				anotherApp        *parser.App
				anotherDesiredLRP *models.DesiredLRP

				anotherProcessGuid string

				cellId0, cellId1     string
				instance0, instance1 string

				runningInstance0, runningInstance1, crashingInstance0 diagnosis.InstanceInfo
				expectedActualLRPGroups1, expectedActualLRPGroups2    []*models.ActualLRPGroup
			)

			BeforeEach(func() {
				cellId0 = "cell-id-0"
				cellId1 = "cell-id-1"

				instance0 = "instance-guid-0"
				instance1 = "instance-guid-1"

				anotherProcessGuid = "another-process-guid"

				anotherApp = &parser.App{
					Name: "app2",
					Guid: "app-guid-2",
				}

				anotherDesiredLRP = &models.DesiredLRP{
					ProcessGuid: anotherProcessGuid,
					Instances:   2,
				}

				expectedActualLRPGroups1 = []*models.ActualLRPGroup{
					actualLRPInfo(processGuid, instance0, "RUNNING", cellId0, 0),
				}

				expectedActualLRPGroups2 = []*models.ActualLRPGroup{
					actualLRPInfo(anotherProcessGuid, instance1, "RUNNING", cellId1, 0),
					actualLRPInfo(anotherProcessGuid, "", "CRASHED", "", 1),
				}

				runningInstance0 = instanceInfo(app.Name, app.Guid, processGuid, instance0, cellId0, "RUNNING", 0)
				runningInstance1 = instanceInfo(anotherApp.Name, anotherApp.Guid, anotherProcessGuid, instance1, cellId1, "RUNNING", 0)
				crashingInstance0 = instanceInfo(anotherApp.Name, anotherApp.Guid, anotherProcessGuid, "", "", "CRASHED", 1)
			})

			It("should return an aggregate of the summaries", func() {
				expectedSummary := diagnosis.Summary{
					InstanceSummary: diagnosis.InstanceSummary{
						Tracked: diagnosis.TrackedSummary{
							Running: 2,
						},
						Untracked: diagnosis.UntrackedSummary{
							Crashed: 1,
						},
					},
					TrackedInstances:   []diagnosis.InstanceInfo{runningInstance0, runningInstance1},
					UntrackedInstances: []diagnosis.InstanceInfo{crashingInstance0},
				}

				appSummary0 := diagnosis.DiagnoseApp(app, *desiredLRPs[0], expectedActualLRPGroups1)
				appSummary1 := diagnosis.DiagnoseApp(anotherApp, *anotherDesiredLRP, expectedActualLRPGroups2)

				aggregate := diagnosis.JoinSummaries(appSummary0, appSummary1)
				Expect(aggregate.InstanceSummary).To(Equal(expectedSummary.InstanceSummary))
				Expect(aggregate.TrackedInstances).To(Equal(expectedSummary.TrackedInstances))
				Expect(aggregate.UntrackedInstances).To(Equal(expectedSummary.UntrackedInstances))
				Expect(aggregate.Timestamp).NotTo(BeTemporally("==", time.Time{}))
			})
		})
	})
})
