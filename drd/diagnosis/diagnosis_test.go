package diagnosis_test

import (
	"time"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/diego-stress-tests/drd/diagnosis"
	"code.cloudfoundry.org/diego-stress-tests/drd/parser"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Diagnosis", func() {
	instanceInfo := func(app, appGuid, processGuid, instanceGuid, cellId, state string, index int32) *diagnosis.InstanceInfo {
		return &diagnosis.InstanceInfo{
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

	Describe("DiagnoseProcessGuid", func() {
		var (
			app         *parser.App
			desiredLRPs []*models.DesiredLRP
			processGuid string
		)

		BeforeEach(func() {
			processGuid = "app-guid-process-guid"
			app = &parser.App{
				Name: "app",
				Guid: "app-guid",
			}

			desiredLRPs = []*models.DesiredLRP{
				&models.DesiredLRP{
					ProcessGuid: processGuid,
					Instances:   2,
				},
			}
		})

		Context("DiscoverActualLRPs", func() {
			var (
				expectedActualLRPGroups []*models.ActualLRPGroup
				cellId                  string
				instance0               string
				instance1               string

				runningInstance   *diagnosis.InstanceInfo
				unclaimedInstance *diagnosis.InstanceInfo
				crashedInstance   *diagnosis.InstanceInfo
				claimedInstance   *diagnosis.InstanceInfo

				appSummary diagnosis.Summary
			)

			Context("with actual lrp instances", func() {
				BeforeEach(func() {
					cellId = "cell-id"
					instance0 = "instance-guid-0"
					instance1 = "instance-guid-1"

					desiredLRPs = []*models.DesiredLRP{
						&models.DesiredLRP{
							ProcessGuid: processGuid,
							Instances:   4,
						},
						&models.DesiredLRP{
							ProcessGuid: "not-app-guid-another-process-guid",
							Instances:   1,
						},
					}

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
				})

				It("collects app summary based on desired and actual lrp", func() {
					appSummary = diagnosis.DiagnoseApp(app, *desiredLRPs[0], expectedActualLRPGroups)

					Expect(appSummary.InstanceSummary.Tracked.Running).To(Equal(1))
					Expect(appSummary.InstanceSummary.Tracked.Claimed).To(Equal(1))
					Expect(appSummary.InstanceSummary.Untracked.Unclaimed).To(Equal(1))
					Expect(appSummary.InstanceSummary.Untracked.Crashed).To(Equal(1))

					Expect(appSummary.TrackedInstances).To(ContainElement(runningInstance))
					Expect(appSummary.TrackedInstances).To(ContainElement(claimedInstance))
					Expect(appSummary.UntrackedInstances).To(ContainElement(unclaimedInstance))
					Expect(appSummary.UntrackedInstances).To(ContainElement(crashedInstance))
				})
			})

			Context("when there are missing instances", func() {
				var (
					untrackedInstance        *diagnosis.InstanceInfo
					anotherUntrackedInstance *diagnosis.InstanceInfo
				)

				BeforeEach(func() {
					desiredLRP := &models.DesiredLRP{
						ProcessGuid: processGuid,
						Instances:   3,
					}

					anotherDesiredLRP := &models.DesiredLRP{
						ProcessGuid: "another-guid-another-process-guid",
						Instances:   2,
					}

					anotherApp := &parser.App{
						Name: "another-app",
						Guid: "another-guid",
					}

					expectedActualLRPGroups := []*models.ActualLRPGroup{
						actualLRPInfo(processGuid, instance0, "RUNNING", cellId, 0),
						actualLRPInfo(processGuid, instance1, "RUNNING", cellId, 2),
					}

					anotherExpectedActualLRPGroups := []*models.ActualLRPGroup{
						actualLRPInfo(processGuid, instance0, "RUNNING", cellId, 0),
					}

					untrackedInstance = instanceInfo(app.Name, app.Guid, "", "", "", "MISSING", 1)
					appSummary1 := diagnosis.DiagnoseApp(app, *desiredLRP, expectedActualLRPGroups)

					anotherUntrackedInstance = instanceInfo(anotherApp.Name, anotherApp.Guid, "", "", "", "MISSING", 1)
					appSummary2 := diagnosis.DiagnoseApp(anotherApp, *anotherDesiredLRP, anotherExpectedActualLRPGroups)

					appSummary = diagnosis.JoinSummaries(appSummary1, appSummary2)
				})

				It("diagnoses the missing instance", func() {
					Expect(appSummary.InstanceSummary.Tracked.Running).To(Equal(3))
					Expect(appSummary.InstanceSummary.Untracked.Missing).To(Equal(2))

					Expect(appSummary.UntrackedInstances).To(ContainElement(untrackedInstance))
					Expect(appSummary.UntrackedInstances).To(ContainElement(anotherUntrackedInstance))
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

				runningInstance0, runningInstance1, crashingInstance0 *diagnosis.InstanceInfo
				expectedActualLRPGroups1, expectedActualLRPGroups2    []*models.ActualLRPGroup
			)

			BeforeEach(func() {
				cellId0 = "cell-id-0"
				cellId1 = "cell-id-1"

				instance0 = "instance-guid-0"
				instance1 = "instance-guid-1"

				desiredLRPs = []*models.DesiredLRP{
					&models.DesiredLRP{
						ProcessGuid: processGuid,
						Instances:   1,
					},
				}

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
					TrackedInstances:   []*diagnosis.InstanceInfo{runningInstance0, runningInstance1},
					UntrackedInstances: []*diagnosis.InstanceInfo{crashingInstance0},
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

	Describe("FindInstance", func() {
		var (
			summary                                      diagnosis.Summary
			app, guid, processGuid, instanceGuid, cellId string
			runningInstance0                             *diagnosis.InstanceInfo
		)

		BeforeEach(func() {
			app, guid, processGuid, instanceGuid, cellId = "app0", "guid0", "processguid0", "instance0", "cellId0"
			runningInstance0 = instanceInfo(app, guid, processGuid, instanceGuid, cellId, "RUNNING", 0)
			trackedSummary := diagnosis.Summary{
				InstanceSummary: diagnosis.InstanceSummary{
					Tracked: diagnosis.TrackedSummary{
						Running: 1,
					},
				},
				TrackedInstances: []*diagnosis.InstanceInfo{runningInstance0},
			}
			summary = diagnosis.JoinSummaries(summary, trackedSummary)
		})

		Context("when the instance is not in the TrackedInstances", func() {
			It("returns nil", func() {
				Expect(summary.FindInstance("instance-guid-not-exist")).To(BeNil())
			})
		})

		Context("when the instance is in the TrackedInstances", func() {
			It("returns the instance info", func() {
				Expect(summary.FindInstance(instanceGuid)).To(Equal(runningInstance0))
			})
		})
	})

	Describe("Update", func() {
		var (
			summary diagnosis.Summary
		)

		BeforeEach(func() {
			summary = diagnosis.Summary{
				Timestamp:          time.Now(),
				InstanceSummary:    diagnosis.InstanceSummary{},
				TrackedInstances:   []*diagnosis.InstanceInfo{},
				UntrackedInstances: []*diagnosis.InstanceInfo{},
			}
		})

		It("only updates state when a process guid is already known", func() {
			actualLRP := actualLRPInfo("guid", "instance0", "RUNNING", "cell-id", 0)

			Expect(summary.Update(actualLRP.Instance.GetInstanceGuid(), actualLRP.Instance.GetState(), "")).To(BeFalse())
		})

		Context("when an instance guid is being tracked", func() {
			var (
				app, guid, processGuid, instanceGuid, cellId string
			)

			Context("and the tracked instance is RUNNING", func() {
				BeforeEach(func() {
					app, guid, processGuid, instanceGuid, cellId = "app0", "guid0", "processguid0", "instance0", "cellId0"
					runningInstance0 := instanceInfo(app, guid, processGuid, instanceGuid, cellId, "RUNNING", 0)
					trackedSummary := diagnosis.Summary{
						InstanceSummary: diagnosis.InstanceSummary{
							Tracked: diagnosis.TrackedSummary{
								Running: 1,
							},
						},
						TrackedInstances: []*diagnosis.InstanceInfo{runningInstance0},
					}
					summary = diagnosis.JoinSummaries(summary, trackedSummary)
				})

				Context("and the instance is in the same state", func() {
					It("does nothing", func() {
						actualLRP := actualLRPInfo(processGuid, instanceGuid, "RUNNING", cellId, 0)
						Expect(summary.Update(actualLRP.Instance.GetInstanceGuid(), actualLRP.Instance.GetState(), "")).To(BeFalse())
					})
				})

				Context("and the instance crashed", func() {
					var actualLRP *models.ActualLRPGroup
					BeforeEach(func() {
						actualLRP = actualLRPInfo(processGuid, instanceGuid, "CRASHED", cellId, 0)
					})

					It("updates the TrackedInstances field", func() {
						Expect(summary.Update(actualLRP.Instance.GetInstanceGuid(), actualLRP.Instance.GetState(), "i crashed")).To(BeTrue())

						info := summary.FindInstance(actualLRP.Instance.InstanceGuid)
						Expect(info).ToNot(BeNil())
						Expect(info.State).To(Equal("CRASHED"))
						Expect(info.CrashReason).ToNot(BeEmpty())
					})

					It("increments the observed crashes field", func() {
						Expect(summary.Update(actualLRP.Instance.GetInstanceGuid(), actualLRP.Instance.GetState(), "")).To(BeTrue())
						Expect(summary.InstanceSummary.Tracked.ObservedCrashes).To(Equal(1))
					})

					It("decrements the running field", func() {
						Expect(summary.Update(actualLRP.Instance.GetInstanceGuid(), actualLRP.Instance.GetState(), "")).To(BeTrue())
						Expect(summary.InstanceSummary.Tracked.Running).To(Equal(0))
					})
				})

				Context("and the instance moves to a CLAIMED state", func() {
					var actualLRP *models.ActualLRPGroup
					BeforeEach(func() {
						actualLRP = actualLRPInfo(processGuid, instanceGuid, "CLAIMED", cellId, 0)
					})

					It("updates the TrackedInstances field", func() {
						Expect(summary.Update(actualLRP.Instance.GetInstanceGuid(), actualLRP.Instance.GetState(), "")).To(BeTrue())

						info := summary.FindInstance(actualLRP.Instance.InstanceGuid)
						Expect(info).ToNot(BeNil())
						Expect(info.State).To(Equal("CLAIMED"))
					})

					It("increments the claimed field", func() {
						Expect(summary.Update(actualLRP.Instance.GetInstanceGuid(), actualLRP.Instance.GetState(), "")).To(BeTrue())
						Expect(summary.InstanceSummary.Tracked.Claimed).To(Equal(1))
					})

					It("does not change the observed crashes field", func() {
						Expect(summary.Update(actualLRP.Instance.GetInstanceGuid(), actualLRP.Instance.GetState(), "")).To(BeTrue())
						Expect(summary.InstanceSummary.Tracked.ObservedCrashes).To(Equal(0))
					})

					It("decrements the running field", func() {
						Expect(summary.Update(actualLRP.Instance.GetInstanceGuid(), actualLRP.Instance.GetState(), "")).To(BeTrue())
						Expect(summary.InstanceSummary.Tracked.Running).To(Equal(0))
					})
				})
			})

			Context("and the tracked instance is CLAIMED", func() {
				BeforeEach(func() {
					app, guid, processGuid, instanceGuid, cellId = "app0", "guid0", "processguid0", "instance0", "cellId0"
					runningInstance0 := instanceInfo(app, guid, processGuid, instanceGuid, cellId, "CLAIMED", 0)
					trackedSummary := diagnosis.Summary{
						InstanceSummary: diagnosis.InstanceSummary{
							Tracked: diagnosis.TrackedSummary{
								Claimed: 1,
							},
						},
						TrackedInstances: []*diagnosis.InstanceInfo{runningInstance0},
					}
					summary = diagnosis.JoinSummaries(summary, trackedSummary)
				})

				Context("and the instance changed to RUNNING", func() {
					var actualLRP *models.ActualLRPGroup
					BeforeEach(func() {
						actualLRP = actualLRPInfo(processGuid, instanceGuid, "RUNNING", cellId, 0)
					})

					It("updates the TrackedInstances field", func() {
						Expect(summary.Update(actualLRP.Instance.GetInstanceGuid(), actualLRP.Instance.GetState(), "")).To(BeTrue())

						info := summary.FindInstance(actualLRP.Instance.InstanceGuid)
						Expect(info).ToNot(BeNil())
						Expect(info.State).To(Equal("RUNNING"))
					})

					It("increments the running field", func() {
						Expect(summary.Update(actualLRP.Instance.GetInstanceGuid(), actualLRP.Instance.GetState(), "")).To(BeTrue())
						Expect(summary.InstanceSummary.Tracked.Running).To(Equal(1))
					})

					It("decrements the claimed field", func() {
						Expect(summary.Update(actualLRP.Instance.GetInstanceGuid(), actualLRP.Instance.GetState(), "")).To(BeTrue())
						Expect(summary.InstanceSummary.Tracked.Claimed).To(Equal(0))
					})
				})
			})

			Context("and the tracked instance is crashed", func() {
				BeforeEach(func() {
					app, guid, processGuid, instanceGuid, cellId = "app0", "guid0", "processguid0", "instance0", "cellId0"
					runningInstance0 := instanceInfo(app, guid, processGuid, instanceGuid, cellId, "CLAIMED", 0)
					trackedSummary := diagnosis.Summary{
						InstanceSummary: diagnosis.InstanceSummary{
							Tracked: diagnosis.TrackedSummary{
								Claimed: 1,
							},
						},
						TrackedInstances: []*diagnosis.InstanceInfo{runningInstance0},
					}
					summary = diagnosis.JoinSummaries(summary, trackedSummary)
				})

				Context("and the instance changed to RUNNING", func() {
					var actualLRP *models.ActualLRPGroup
					BeforeEach(func() {
						actualLRP = actualLRPInfo(processGuid, instanceGuid, "CRASHED", cellId, 0)
					})

					It("updates the TrackedInstances field", func() {
						Expect(summary.Update(actualLRP.Instance.GetInstanceGuid(), actualLRP.Instance.GetState(), "")).To(BeTrue())

						info := summary.FindInstance(actualLRP.Instance.InstanceGuid)
						Expect(info).ToNot(BeNil())
						Expect(info.State).To(Equal("CRASHED"))
					})

					It("increments the running field", func() {
						Expect(summary.Update(actualLRP.Instance.GetInstanceGuid(), actualLRP.Instance.GetState(), "")).To(BeTrue())
						Expect(summary.InstanceSummary.Tracked.ObservedCrashes).To(Equal(1))
					})

					It("decrements the claimed field", func() {
						Expect(summary.Update(actualLRP.Instance.GetInstanceGuid(), actualLRP.Instance.GetState(), "")).To(BeTrue())
						Expect(summary.InstanceSummary.Tracked.Claimed).To(Equal(0))
					})
				})
			})
		})
	})
})
