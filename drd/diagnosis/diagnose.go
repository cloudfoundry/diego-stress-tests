package diagnosis

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"time"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/diego-stress-tests/drd/parser"
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

	for _, actualLRP := range actualLRPs {
		switch actualLRP.Instance.State {
		case "RUNNING":
			summary.InstanceSummary.Tracked.Running++
			collectTrackedInfo(app, actualLRP, &summary)
		case "CLAIMED":
			summary.InstanceSummary.Tracked.Claimed++
			collectTrackedInfo(app, actualLRP, &summary)
		case "CRASHED":
			summary.InstanceSummary.Untracked.Crashed++
			collectUntrackedInfo(app, actualLRP, &summary)
		case "UNCLAIMED":
			summary.InstanceSummary.Untracked.Unclaimed++
			collectUntrackedInfo(app, actualLRP, &summary)
			// TODO: count missing and observed crashes
		}
	}
	return summary
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

func WriteToFile(summary Summary, filePath string) error {
	summaryBytes, err := json.Marshal(summary)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filePath, summaryBytes, 0644)
	if err != nil {
		return err
	}
	return nil
}
