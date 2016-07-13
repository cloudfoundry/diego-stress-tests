package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/lager/chug"
)

type Mapper interface {
	Run(in <-chan chug.Entry, metrics chan<- Metric)
}

type Metric struct {
	Name      string
	Tags      []string
	Value     string
	Timestamp time.Time
}

type requestLatencyMapper struct {
}

func NewRequestLatencyMapper() Mapper {
	return requestLatencyMapper{}
}

func (m requestLatencyMapper) Run(in <-chan chug.Entry, metrics chan<- Metric) {
	mapper := make(map[string]chug.Entry)

	for entry := range in {
		key, err := m.getKey(entry)
		if err != nil {
			continue
		}

		if strings.Contains(entry.Log.Message, "request.serving") {
			mapper[key] = entry
		}

		if strings.Contains(entry.Log.Message, "request.done") {
			if servingEntry, ok := mapper[key]; ok {
				timeDiff := entry.Log.Timestamp.Sub(servingEntry.Log.Timestamp)
				metrics <- Metric{
					Name:      "RequestLatency",
					Tags:      []string{fmt.Sprintf("request=%s", entry.Log.Data["request"])},
					Value:     strconv.FormatInt(int64(timeDiff), 10),
					Timestamp: servingEntry.Log.Timestamp,
				}
				delete(mapper, key)
			}
		}
	}
}

func (m requestLatencyMapper) getKey(entry chug.Entry) (string, error) {
	if entry.Log.Data["request"] == nil {
		return "", fmt.Errorf("not an http request")
	}
	return fmt.Sprintf("%s:%s", entry.Log.Data["request"], entry.Log.Session), nil
}
