package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"code.cloudfoundry.org/lager/chug"
)

const metricPrefix = "cf.diego."

func main() {
	flag.Parse()

	chugOut := make(chan chug.Entry)
	go chug.Chug(os.Stdin, chugOut)

	metrics := make(chan Metric)

	go func() {
		for metric := range metrics {
			tags := mapToTags(metric.Tags)
			fmt.Printf("%s%s,%s value=%s %d\n",
				metricPrefix,
				metric.Name,
				strings.Join(tags, ","),
				metric.Value,
				metric.Timestamp.UnixNano(),
			)
		}
	}()

	mapAll(chugOut, metrics,
		RequestLatencyMapper,
		AuctionSchedulingMapper,
		TaskLifecycleMapper,
		LRPLifecycleMapper,
	)
}

func mapToTags(m map[string]string) []string {
	tags := make([]string, 0, len(m))
	for key, value := range m {
		tags = append(tags, fmt.Sprintf("%s=%s", key, value))
	}
	return tags
}
