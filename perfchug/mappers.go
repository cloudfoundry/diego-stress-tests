package main

import (
	"fmt"
	"strconv"
	"strings"

	"code.cloudfoundry.org/lager/chug"
)

var RequestLatencyMapper = &Mapper{
	Name: "RequestLatencyMapper",

	StartString: "request.serving",
	EndString:   "request.done",

	Transform: func(s, e chug.Entry) Metric {
		component := strings.Split(e.Log.Message, ".")[0]
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		return Metric{
			Name: "RequestLatency",
			Tags: map[string]string{
				"component": component,
				"request":   fmt.Sprint(e.Log.Data["request"]),
			},
			Value:     strconv.FormatInt(int64(timeDiff), 10),
			Timestamp: s.Log.Timestamp,
		}
	},

	GetKey: func(entry chug.Entry) (string, error) {
		if entry.Log.Data["request"] == nil {
			return "", fmt.Errorf("not an http request")
		}
		return fmt.Sprintf("%s:%s", entry.Log.Data["request"], entry.Log.Session), nil
	},

	entriesMap: make(map[string]chug.Entry),
}

var AuctionSchedulingMapper = &Mapper{
	Name: "AuctionSchedulingMapper",

	StartString: "auction.scheduling",
	EndString:   "auction.scheduled",

	Transform: func(s, e chug.Entry) Metric {
		component := strings.Split(e.Log.Message, ".")[0]
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		return Metric{
			Name: "AuctionScheduleDuration",
			Tags: map[string]string{
				"component": component,
			},
			Value:     strconv.FormatInt(int64(timeDiff), 10),
			Timestamp: s.Log.Timestamp,
		}
	},

	GetKey: func(entry chug.Entry) (string, error) {
		return entry.Log.Session, nil
	},

	entriesMap: make(map[string]chug.Entry),
}

var TaskLifecycleMapper = &Mapper{
	Name: "TaskLifecycleMapper",

	StartString: "desire-task.starting",
	EndString:   "complete-task.complete",

	Transform: func(s, e chug.Entry) Metric {
		component := strings.Split(e.Log.Message, ".")[0]
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		return Metric{
			Name: "TaskLifecycle",
			Tags: map[string]string{
				"component": component,
				"task_guid": fmt.Sprint(e.Log.Data["task_guid"]),
			},
			Value:     strconv.FormatInt(int64(timeDiff), 10),
			Timestamp: s.Log.Timestamp,
		}
	},

	GetKey: func(entry chug.Entry) (string, error) {
		if entry.Log.Data["task_guid"] == nil {
			return "", fmt.Errorf("not a task log line")
		}
		return fmt.Sprint(entry.Log.Data["task_guid"]), nil
	},

	entriesMap: make(map[string]chug.Entry),
}

var LRPLifecycleMapper = &Mapper{
	Name: "LRPLifecycleMapper",

	StartString: "create-unclaimed-actual-lrp.starting",
	EndString:   "start-actual-lrp.complete",

	Transform: func(s, e chug.Entry) Metric {
		component := strings.Split(e.Log.Message, ".")[0]
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		keyData := e.Log.Data["actual_lrp_key"].(map[string]interface{})
		return Metric{
			Name: "LRPLifecycle",
			Tags: map[string]string{
				"component":    component,
				"process_guid": fmt.Sprint(keyData["process_guid"]),
				"index":        fmt.Sprint(keyData["index"]),
			},
			Value:     strconv.FormatInt(int64(timeDiff), 10),
			Timestamp: s.Log.Timestamp,
		}
	},

	GetKey: func(entry chug.Entry) (string, error) {
		if entry.Log.Data["actual_lrp_key"] == nil {
			return "", fmt.Errorf("not an LRP log line")
		}

		keyData := entry.Log.Data["actual_lrp_key"].(map[string]interface{})

		if keyData["process_guid"] == nil {
			return "", fmt.Errorf("not an LRP log line")
		}
		if keyData["index"] == nil {
			return "", fmt.Errorf("not an LRP log line")
		}
		return fmt.Sprintf("%s:%v", keyData["process_guid"], keyData["index"]), nil
	},

	entriesMap: make(map[string]chug.Entry),
}

var CedarSuccessfulPushMapper = &Mapper{
	Name: "CedarSuccessfulPushMapper",

	StartString: "cedar.push.started",
	EndString:   "cedar.push.completed",

	Transform: func(s, e chug.Entry) Metric {
		component := strings.Split(e.Log.Message, ".")[0]
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		return Metric{
			Name: "CedarSuccessfulPush",
			Tags: map[string]string{
				"component": component,
				"app":       fmt.Sprint(e.Log.Data["app"]),
				"session":   fmt.Sprint(e.Log.Data["session"]),
			},
			Value:     strconv.FormatInt(int64(timeDiff), 10),
			Timestamp: s.Log.Timestamp,
		}
	},

	GetKey: func(entry chug.Entry) (string, error) {
		if entry.Log.Data["app"] == nil {
			return "", fmt.Errorf("not a cedar push log line")
		}
		return fmt.Sprint(entry.Log.Data["app"]), nil
	},

	entriesMap: make(map[string]chug.Entry),
}
