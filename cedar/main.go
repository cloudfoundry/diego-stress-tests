package main

import (
	"flag"

	"code.cloudfoundry.org/cflager"
)

var (
	numBatches       = flag.Int("n", 0, "number of batches to seed")
	maxInFlight      = flag.Int("k", 1, "max number of cf operations in flight")
	domain           = flag.String("domain", "bosh-lite.com", "app domain")
	maxPollingErrors = flag.Int("max-polling-errors", 1, "max number of curl failures")
	configFile       = flag.String("config", "config.json", "path to cedar config file")
	appPayload       = flag.String("payload", "assets/temp-app", "directory containing the stress-app payload to push")
	tolerance        = flag.Float64("tolerance", 1.0, "fractional failure tolerance")
)

type appDefinition struct {
	ManifestPath  string `json:"manifestPath"`
	AppNamePrefix string `json:"appNamePrefix"`
	AppCount      int    `json:"appCount"`
}

func main() {
	cflager.AddFlags(flag.CommandLine)

	flag.Parse()

	logger, _ := cflager.New("cedar")
	logger.Info("started")
	defer logger.Info("exited")

	config := Config{
		numBatches:       *numBatches,
		maxInFlight:      *maxInFlight,
		maxPollingErrors: *maxPollingErrors,
		tolerance:        *tolerance,
		appPayload:       *appPayload,
		domain:           *domain,
		configFile:       *configFile,
	}

	config.Init(logger)

	pusher := NewPusher(config)
	pusher.PushApps(logger)
	pusher.StartApps(logger)
}
