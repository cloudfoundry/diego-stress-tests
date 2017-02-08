package main

import (
	"fmt"
	"strconv"
	"strings"

	"code.cloudfoundry.org/lager/chug"
)

var ContainerCreationMapper = &Mapper{
	Name: "ContainerCreationMapper",

	StartString: "run-container.creating-container",
	EndString:   "run-container.succeeded-creating-container-in-garden",

	Transform: func(s, e chug.Entry) Metric {
		component := "executor"
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		return Metric{
			Name: "ContainerCreation",
			Tags: map[string]string{
				"component":      component,
				"container-guid": fmt.Sprintf("%s", e.Log.Data["container-guid"]),
			},
			Value:     strconv.FormatInt(int64(timeDiff), 10),
			Timestamp: s.Log.Timestamp,
		}
	},

	GetKey: func(entry chug.Entry) (string, error) {
		if entry.Log.Data["container-guid"] == nil {
			return "", fmt.Errorf("not an http request")
		}
		return fmt.Sprintf("%s:%s", entry.Log.Data["container-guid"], entry.Log.Session), nil
	},

	entriesMap: make(map[string]chug.Entry),
}

var ContainerRunnerMapper = &Mapper{
	Name: "ContainerRunnerMapper",

	StartString: "run-container.running-container-in-garden",
	EndString:   "run-container.succeeded-running-container-in-garden",

	Transform: func(s, e chug.Entry) Metric {
		component := "executor"
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		return Metric{
			Name: "ContainerRun",
			Tags: map[string]string{
				"component":      component,
				"container-guid": fmt.Sprintf("%s", e.Log.Data["container-guid"]),
			},
			Value:     strconv.FormatInt(int64(timeDiff), 10),
			Timestamp: s.Log.Timestamp,
		}
	},

	GetKey: func(entry chug.Entry) (string, error) {
		if entry.Log.Data["container-guid"] == nil {
			return "", fmt.Errorf("not an http request")
		}
		return fmt.Sprintf("%s:%s", entry.Log.Data["container-guid"], entry.Log.Session), nil
	},

	entriesMap: make(map[string]chug.Entry),
}

var CertIOMapper = &Mapper{
	Name: "CertIOMapper",

	StartString: "save-keys.starting",
	EndString:   "save-keys.done",

	Transform: func(s, e chug.Entry) Metric {
		component := "executor"
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		return Metric{
			Name: "CertIO",
			Tags: map[string]string{
				"component":      component,
				"container-guid": fmt.Sprintf("%s", e.Log.Data["container-guid"]),
			},
			Value:     strconv.FormatInt(int64(timeDiff), 10),
			Timestamp: s.Log.Timestamp,
		}
	},

	GetKey: func(entry chug.Entry) (string, error) {
		if entry.Log.Data["container-guid"] == nil {
			return "", fmt.Errorf("not an http request")
		}
		return fmt.Sprintf("%s:%s", entry.Log.Data["container-guid"], entry.Log.Session), nil
	},

	entriesMap: make(map[string]chug.Entry),
}

var CertGenerationMapper = &Mapper{
	Name: "KeyGenerationMapper",

	StartString: "create-certs.starting",
	EndString:   "create-certs.done",

	Transform: func(s, e chug.Entry) Metric {
		component := "executor"
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		return Metric{
			Name: "CertGeneration",
			Tags: map[string]string{
				"component":      component,
				"container-guid": fmt.Sprintf("%s", e.Log.Data["container-guid"]),
			},
			Value:     strconv.FormatInt(int64(timeDiff), 10),
			Timestamp: s.Log.Timestamp,
		}
	},

	GetKey: func(entry chug.Entry) (string, error) {
		if entry.Log.Data["container-guid"] == nil {
			return "", fmt.Errorf("not an http request")
		}
		return fmt.Sprintf("%s:%s", entry.Log.Data["container-guid"], entry.Log.Session), nil
	},

	entriesMap: make(map[string]chug.Entry),
}

var KeyGenerationMapper = &Mapper{
	Name: "KeyGenerationMapper",

	StartString: "create-keys.starting",
	EndString:   "create-keys.done",

	Transform: func(s, e chug.Entry) Metric {
		component := "executor"
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		return Metric{
			Name: "KeyGeneration",
			Tags: map[string]string{
				"component":      component,
				"container-guid": fmt.Sprintf("%s", e.Log.Data["container-guid"]),
			},
			Value:     strconv.FormatInt(int64(timeDiff), 10),
			Timestamp: s.Log.Timestamp,
		}
	},

	GetKey: func(entry chug.Entry) (string, error) {
		if entry.Log.Data["container-guid"] == nil {
			return "", fmt.Errorf("not an http request")
		}
		return fmt.Sprintf("%s:%s", entry.Log.Data["container-guid"], entry.Log.Session), nil
	},

	entriesMap: make(map[string]chug.Entry),
}

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

	StartString: "cedar.pushing-apps.push.started",
	EndString:   "cedar.pushing-apps.push.completed",

	Transform: func(s, e chug.Entry) Metric {
		component := strings.Split(e.Log.Message, ".")[0]
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		timeMean := s.Log.Timestamp.Add(timeDiff / 2)

		return Metric{
			Name: "CedarSuccessfulPush",
			Tags: map[string]string{
				"component": component,
				"app":       fmt.Sprint(e.Log.Data["app"]),
				"session":   fmt.Sprint(e.Log.Session),
			},
			Value:     strconv.FormatInt(int64(timeDiff), 10),
			Timestamp: timeMean,
		}
	},

	GetKey: func(entry chug.Entry) (string, error) {
		//	fmt.Printf("ENTRY is %s\n", entry)
		if entry.Log.Data["app"] == nil {
			return "", fmt.Errorf("not a cedar push log line")
		}
		return fmt.Sprint(entry.Log.Data["app"]), nil
	},

	entriesMap: make(map[string]chug.Entry),
}

var CedarFailedPushMapper = &Mapper{
	Name: "CedarFailedPushMapper",

	StartString: "cedar.pushing-apps.push.started",
	EndString:   "cedar.pushing-apps.push.failed",

	Transform: func(s, e chug.Entry) Metric {
		component := strings.Split(e.Log.Message, ".")[0]
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		timeMean := s.Log.Timestamp.Add(timeDiff / 2)

		return Metric{
			Name: "CedarFailedPush",
			Tags: map[string]string{
				"component": component,
				"app":       fmt.Sprint(e.Log.Data["app"]),
				"session":   fmt.Sprint(e.Log.Session),
			},
			Value:     strconv.FormatInt(int64(timeDiff), 10),
			Timestamp: timeMean,
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

var CedarSuccessfulStartMapper = &Mapper{
	Name: "CedarSuccessfulStartMapper",

	StartString: "cedar.starting-apps.start.started",
	EndString:   "cedar.starting-apps.start.completed",

	Transform: func(s, e chug.Entry) Metric {
		component := strings.Split(e.Log.Message, ".")[0]
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		timeMean := s.Log.Timestamp.Add(timeDiff / 2)

		return Metric{
			Name: "CedarSuccessfulStart",
			Tags: map[string]string{
				"component": component,
				"app":       fmt.Sprint(e.Log.Data["app"]),
				"session":   fmt.Sprint(e.Log.Session),
			},
			Value:     strconv.FormatInt(int64(timeDiff), 10),
			Timestamp: timeMean,
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

var CedarFailedStartMapper = &Mapper{
	Name: "CedarFailedStartMapper",

	StartString: "cedar.starting-apps.start.started",
	EndString:   "cedar.starting-apps.start.failed",

	Transform: func(s, e chug.Entry) Metric {
		component := strings.Split(e.Log.Message, ".")[0]
		timeDiff := e.Log.Timestamp.Sub(s.Log.Timestamp)
		timeMean := s.Log.Timestamp.Add(timeDiff / 2)

		return Metric{
			Name: "CedarFailedStart",
			Tags: map[string]string{
				"component": component,
				"app":       fmt.Sprint(e.Log.Data["app"]),
				"session":   fmt.Sprint(e.Log.Session),
			},
			Value:     strconv.FormatInt(int64(timeDiff), 10),
			Timestamp: timeMean,
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
