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
			fmt.Printf("%s%s,%s value=%s %d\n", metricPrefix, metric.Name, strings.Join(metric.Tags, ","), metric.Value, metric.Timestamp.UnixNano())
		}
	}()

	NewRequestLatencyMapper().Run(chugOut, metrics)

	close(metrics)
}
