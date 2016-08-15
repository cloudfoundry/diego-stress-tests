package main

import (
	"flag"

	"golang.org/x/net/context"

	"code.cloudfoundry.org/cflager"
)

var (
	numBatches       = flag.Int("n", 0, "number of batches to seed")
	maxInFlight      = flag.Int("k", 1, "max number of cf operations in flight")
	domain           = flag.String("domain", "bosh-lite.com", "app domain")
	maxPollingErrors = flag.Int("max-polling-errors", 1, "max number of curl failures")
	configFile       = flag.String("config", "config.json", "path to cedar config file")
	outputFile       = flag.String("output", "output.json", "path to cedar metric results file")
	appPayload       = flag.String("payload", "assets/temp-app", "directory containing the stress-app payload to push")
	tolerance        = flag.Float64("tolerance", 1.0, "fractional failure tolerance")
	timeout          = flag.Int("timeout", 30, "timeout in seconds")
)

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
		outputFile:       *outputFile,
		timeout:          *timeout,
	}

	config.Init(logger)

	ctx := context.WithValue(context.Background(), "logger", logger)
	ctx, cancel := context.WithCancel(context.Background())

	pusher := NewPusher(config)
	pusher.PushApps(ctx, cancel)
	pusher.StartApps(ctx, cancel)
	pusher.GenerateReport(ctx, cancel)
}
