package main

import (
	"flag"
	"os"

	"golang.org/x/net/context"

	"code.cloudfoundry.org/cflager"
)

var (
	cfAPI             = flag.String("api", "api.bosh-lite.com", "location of the cloud controller api")
	adminUser         = flag.String("admin-user", "admin", "uaa admin user")
	adminPassword     = flag.String("admin-password", "admin", "uaa admin password")
	skipSSLValidation = flag.Bool("skip-ssl-validation", true, "skip ssl validation")

	orgName   = flag.String("org", "stress-tests-org", "organization to use for stress tests")
	spaceName = flag.String("space", "stress-tests-space", "space to use for stress tests")

	domain = flag.String("domain", "bosh-lite.com", "app domain")

	numBatches       = flag.Int("n", 1, "number of batches to seed")
	maxInFlight      = flag.Int("k", 1, "max number of cf operations in flight")
	maxPollingErrors = flag.Int("max-polling-errors", 1, "max number of curl failures")
	tolerance        = flag.Float64("tolerance", 1.0, "fractional failure tolerance")
	configFile       = flag.String("config", "config.json", "path to cedar config file")
	outputFile       = flag.String("output", "output.json", "path to cedar metric results file")
	appPayload       = flag.String("payload", "assets/temp-app", "directory containing the stress-app payload to push")
	timeout          = flag.Int("timeout", 30, "time allowed for a push or start operation , in seconds")
)

func main() {
	cflager.AddFlags(flag.CommandLine)

	flag.Parse()

	logger, _ := cflager.New("cedar")
	logger.Info("started")
	defer logger.Info("exited")

	config := Config{
		NumBatches:       *numBatches,
		MaxInFlight:      *maxInFlight,
		MaxPollingErrors: *maxPollingErrors,
		Tolerance:        *tolerance,
		AppPayload:       *appPayload,
		Domain:           *domain,
		ConfigFile:       *configFile,
		OutputFile:       *outputFile,
		Timeout:          *timeout,
	}

	config.Init(logger)

	ctx, cancel := context.WithCancel(
		context.WithValue(
			context.Background(),
			"logger",
			logger,
		),
	)

	ctxWithTimeout, _ := context.WithTimeout(ctx, config.TimeoutDuration())
	err := setupCFCLI(ctxWithTimeout)
	if err != nil {
		logger.Error("failed-to-setup-cf", err)
		os.Exit(1)
	}

	apps := NewAppGenerator(config).Apps(ctx)

	pusher := NewPusher(config, apps)
	pusher.PushApps(ctx, cancel)
	pusher.StartApps(ctx, cancel)
	pusher.GenerateReport(ctx, cancel)
}
