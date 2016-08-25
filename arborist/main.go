package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/diego-stress-tests/arborist/parser"
	"code.cloudfoundry.org/diego-stress-tests/arborist/watcher"
	"code.cloudfoundry.org/lager"
)

var (
	requestInterval = flag.Int("request-interval", 60, "interval in seconds at which to make requests to each individual app")
	duration        = flag.Int("duration", 600, "total duration to check routability of applications")
	appFile         = flag.String("app-file", "", "path to json application file")
	resultFile      = flag.String("result-file", "output.json", "path to result file")
	domain          = flag.String("domain", "bosh-lite.com", "domain where the applications are deployed")
)

func main() {
	cflager.AddFlags(flag.CommandLine)

	flag.Parse()

	logger, _ := cflager.New("arborist")
	clock := clock.NewClock()

	validateFlags(logger)

	logger.Info("started")
	defer logger.Info("exited")

	applications, err := parser.ParseAppFile(logger, *appFile, *domain)
	if err != nil {
		logger.Error("failed-to-parse-app-file", err)
		os.Exit(1)
	}

	results, err := watcher.CheckRoutability(logger, clock, applications, *duration, *requestInterval)
	if err != nil {
		// This should be impossible
		logger.Error("failed-to-check-routability", err)
		os.Exit(1)
	}

	resultJSON, err := json.Marshal(results)
	if err != nil {
		logger.Error("failed-to-marshal-result-json", err)
		os.Exit(1)
	}

	err = ioutil.WriteFile(*resultFile, resultJSON, 0644)
	if err != nil {
		logger.Error("failed-to-write-result-file", err, lager.Data{"result": resultJSON})
		os.Exit(1)
	}
}

func validateFlags(logger lager.Logger) {
	validationErr := errors.New("validation-error")

	if *appFile == "" {
		logger.Error("app-file must be specified", validationErr)
		os.Exit(1)
	}

	if *resultFile == "" {
		logger.Error("result-file must be specified", validationErr)
		os.Exit(1)
	}

	if *domain == "" {
		logger.Error("domain must be specified", validationErr)
		os.Exit(1)
	}

	if *duration <= 0 {
		logger.Error("duration must be greater than 0", validationErr)
		os.Exit(1)
	}

	if *requestInterval <= 0 {
		logger.Error("interval must be greater than 0", validationErr)
		os.Exit(1)
	}
}
