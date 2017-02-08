package main

import (
	"flag"
	"fmt"
	"os"

	"code.cloudfoundry.org/lager/chug"
)

const metricPrefix = "cf.diego."

type CSVMetric struct {
	Guid            string
	ContainerCreate string
	ContainerRun    string
	KeyCreate       string
	CertCreate      string
	IOSave          string
}

func main() {
	flag.Parse()

	chugOut := make(chan chug.Entry, 1)
	go chug.Chug(os.Stdin, chugOut)

	metrics := make(chan Metric)

	data := map[string]*CSVMetric{}

	go func() {
		for metric := range metrics {
			guid := metric.Tags["container-guid"]
			m, ok := data[guid]
			if !ok {
				m = &CSVMetric{Guid: guid}
			}

			switch metric.Name {
			case "ContainerCreation":
				m.ContainerCreate = metric.Value
			case "ContainerRun":
				m.ContainerRun = metric.Value
			case "CertIO":
				m.IOSave = metric.Value
			case "KeyGeneration":
				m.KeyCreate = metric.Value
			case "CertGeneration":
				m.CertCreate = metric.Value
			default:
			}

			data[guid] = m
		}
	}()

	mapAll(chugOut, metrics,
		ContainerCreationMapper,
		ContainerRunnerMapper,
		KeyGenerationMapper,
		CertGenerationMapper,
		CertIOMapper,
		// RequestLatencyMapper,
		// AuctionSchedulingMapper,
		// TaskLifecycleMapper,
		// LRPLifecycleMapper,
		// CedarSuccessfulPushMapper,
		// CedarFailedPushMapper,
		// CedarSuccessfulStartMapper,
		// CedarFailedStartMapper,
	)

	for _, m := range data {
		fmt.Printf("%s,%s,%s,%s,%s,%s\n", m.Guid, m.ContainerCreate, m.KeyCreate,
			m.CertCreate, m.IOSave, m.ContainerRun)
	}
}

func mapToTags(m map[string]string) []string {
	tags := make([]string, 0, len(m))
	for key, value := range m {
		tags = append(tags, fmt.Sprintf("%s=%s", key, value))
	}
	return tags
}
