package diagnosis

import (
	"strings"
	"time"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/diego-stress-tests/drd/parser"
)

const (
	ActualLRPStateMissing = "MISSING"
)

type TrackedSummary struct {
	Claimed         int `json:"claimed"`
	Running         int `json:"running"`
	ObservedCrashes int `json:"observed_crashes"`
}

type UntrackedSummary struct {
	Unclaimed int `json:"unclaimed"`
	Crashed   int `json:"crashed"`
	Missing   int `json:"missing"`
}

type InstanceSummary struct {
	Tracked   TrackedSummary   `json:"tracked"`
	Untracked UntrackedSummary `json:"untracked"`
}

type InstanceInfo struct {
	InstanceGuid string `json:"instance_guid,omitempty"`
	CellId       string `json:"cell_id,omitempty"`
	AppName      string `json:"app_name"`
	AppGuid      string `json:"app_guid"`
	ProcessGuid  string `json:"process_guid"`
	Index        int32  `json:"index"`
	State        string `json:"state"`
}

type Summary struct {
	Timestamp          time.Time       `json:"timestamp"`
	InstanceSummary    InstanceSummary `json:"instance_summary"`
	TrackedInstances   []InstanceInfo  `json:"tracked_instances"`
	UntrackedInstances []InstanceInfo  `json:"untracked_instances"`
}

func DiscoverProcessGuid(app *parser.App, desiredLRPs []*models.DesiredLRP) *models.DesiredLRP {
	for _, desiredLRP := range desiredLRPs {
		if strings.HasPrefix(desiredLRP.ProcessGuid, app.Guid) {
			return desiredLRP
		}
	}
	return nil
}

func DiagnoseApp(app *parser.App, desiredLRP models.DesiredLRP, actualLRPs []*models.ActualLRPGroup) Summary {
	summary := Summary{
		Timestamp: time.Now(),
		InstanceSummary: InstanceSummary{
			Tracked:   TrackedSummary{},
			Untracked: UntrackedSummary{},
		},
		TrackedInstances:   []InstanceInfo{},
		UntrackedInstances: []InstanceInfo{},
	}

	// TODO: count observed crashes
	for _, actualLRP := range actualLRPs {
		switch actualLRP.Instance.State {
		case models.ActualLRPStateRunning:
			summary.InstanceSummary.Tracked.Running++
			collectTrackedInfo(app, actualLRP, &summary)
		case models.ActualLRPStateClaimed:
			summary.InstanceSummary.Tracked.Claimed++
			collectTrackedInfo(app, actualLRP, &summary)
		case models.ActualLRPStateCrashed:
			summary.InstanceSummary.Untracked.Crashed++
			collectUntrackedInfo(app, actualLRP, &summary)
		case models.ActualLRPStateUnclaimed:
			summary.InstanceSummary.Untracked.Unclaimed++
			collectUntrackedInfo(app, actualLRP, &summary)
		}
	}

	missingInstanceCount := int(desiredLRP.Instances) - len(actualLRPs)
	if missingInstanceCount > 0 {
		summary.InstanceSummary.Untracked.Missing += missingInstanceCount
		collectMissingInfo(app, desiredLRP, actualLRPs, &summary)
	}
	return summary
}

func JoinSummaries(summary1, summary2 Summary) Summary {
	aggregate := Summary{
		Timestamp: time.Now(),
		InstanceSummary: InstanceSummary{
			Tracked: TrackedSummary{
				Claimed:         summary1.InstanceSummary.Tracked.Claimed + summary2.InstanceSummary.Tracked.Claimed,
				Running:         summary1.InstanceSummary.Tracked.Running + summary2.InstanceSummary.Tracked.Running,
				ObservedCrashes: summary1.InstanceSummary.Tracked.ObservedCrashes + summary2.InstanceSummary.Tracked.ObservedCrashes,
			},
			Untracked: UntrackedSummary{
				Unclaimed: summary1.InstanceSummary.Untracked.Unclaimed + summary2.InstanceSummary.Untracked.Unclaimed,
				Crashed:   summary1.InstanceSummary.Untracked.Crashed + summary2.InstanceSummary.Untracked.Crashed,
				Missing:   summary1.InstanceSummary.Untracked.Missing + summary2.InstanceSummary.Untracked.Missing,
			},
		},
	}

	aggregate.TrackedInstances = append(summary1.TrackedInstances, summary2.TrackedInstances...)
	aggregate.UntrackedInstances = append(summary1.UntrackedInstances, summary2.UntrackedInstances...)
	return aggregate
}

func collectTrackedInfo(app *parser.App, actualLRP *models.ActualLRPGroup, summary *Summary) {
	instanceInfo := InstanceInfo{
		InstanceGuid: actualLRP.Instance.ActualLRPInstanceKey.InstanceGuid,
		CellId:       actualLRP.Instance.ActualLRPInstanceKey.CellId,
		ProcessGuid:  actualLRP.Instance.ActualLRPKey.ProcessGuid,
		Index:        actualLRP.Instance.ActualLRPKey.Index,
		State:        actualLRP.Instance.State,
		AppName:      app.Name,
		AppGuid:      app.Guid,
	}
	summary.TrackedInstances = append(summary.TrackedInstances, instanceInfo)
}

func collectUntrackedInfo(app *parser.App, actualLRP *models.ActualLRPGroup, summary *Summary) {
	instanceInfo := InstanceInfo{
		ProcessGuid: actualLRP.Instance.ActualLRPKey.ProcessGuid,
		Index:       actualLRP.Instance.ActualLRPKey.Index,
		State:       actualLRP.Instance.State,
		AppName:     app.Name,
		AppGuid:     app.Guid,
	}
	summary.UntrackedInstances = append(summary.UntrackedInstances, instanceInfo)
}

func collectMissingInfo(app *parser.App, desiredLRP models.DesiredLRP, actualLRPs []*models.ActualLRPGroup, summary *Summary) {
	actualLRPMap := make(map[int32]*models.ActualLRPGroup)
	for _, actualLRP := range actualLRPs {
		actualLRPMap[actualLRP.Instance.Index] = actualLRP
	}

	for index := int32(0); index < desiredLRP.Instances; index++ {
		actual := actualLRPMap[index]
		if actual != nil {
			continue
		}

		missingInstance := InstanceInfo{
			Index:   index,
			State:   ActualLRPStateMissing,
			AppName: app.Name,
			AppGuid: app.Guid,
		}
		summary.UntrackedInstances = append(summary.UntrackedInstances, missingInstance)
	}
}
