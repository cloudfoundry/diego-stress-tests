package main

import (
	"fmt"
	"strconv"
	"strings"

	"code.cloudfoundry.org/lager/chug"
)

var RequestLatencyMapper = Mapper{
	StartString: "request.serving",
	EndString:   "request.done",

	Transform: func(s, e chug.Entry) Metric {
		component := strings.Split(e.Log.Message, ".")[0]
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		return Metric{
			Name: "RequestLatency",
			Tags: map[string]string{
				"request":   fmt.Sprint(e.Log.Data["request"]),
				"component": component,
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

var AuctionSchedulingMapper = Mapper{
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

var TaskLifecycleMapper = Mapper{
	StartString: "desire-task.starting",
	EndString:   "complete-task.complete",

	Transform: func(s, e chug.Entry) Metric {
		component := strings.Split(e.Log.Message, ".")[0]
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		return Metric{
			Name: "TaskLifecycleMapper",
			Tags: map[string]string{
				"task_guid": fmt.Sprint(e.Log.Data["task_guid"]),
				"component": component,
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

var LRPLifecycleMapper = Mapper{
	StartString: "desire-lrp.starting",
	EndString:   "desire-lrp.complete",

	Transform: func(s, e chug.Entry) Metric {
		component := strings.Split(e.Log.Message, ".")[0]
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		return Metric{
			Name: "LRPLifecycleMapper",
			Tags: map[string]string{
				"process_guid": fmt.Sprint(e.Log.Data["process_guid"]),
				"index":        fmt.Sprint(e.Log.Data["index"]),
				"component":    component,
			},
			Value:     strconv.FormatInt(int64(timeDiff), 10),
			Timestamp: s.Log.Timestamp,
		}
	},

	GetKey: func(entry chug.Entry) (string, error) {
		if entry.Log.Data["process_guid"] == nil {
			return "", fmt.Errorf("not an LRP log line")
		}
		if entry.Log.Data["index"] == nil {
			return "", fmt.Errorf("not an LRP log line")
		}
		return fmt.Sprintf("%s:%d", entry.Log.Data["process_guid"], entry.Log.Data["index"]), nil
	},

	entriesMap: make(map[string]chug.Entry),
}
