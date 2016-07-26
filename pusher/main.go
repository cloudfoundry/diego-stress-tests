package main

import (
	"flag"
	"os"
	"time"

	"golang.org/x/net/context"

	"code.cloudfoundry.org/cflager"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
)

const (
	CFPushTimeout    = 30 * time.Minute
	CFDefaultTimeout = 30 * time.Second
)

var (
	pushRetries = flag.Int("push-retries", 3, "maximum number of tries for a single push")
	batches     = flag.Int("batches", 10, "number of batches of pushes to run")
	batchSize   = flag.Int("batch-size", 10, "size of each parallel batch of pushes")
	appPath     = flag.String("app-path", "../assets/stress-app", "location of the stress app")

	cfAPI             = flag.String("api", "api.bosh-lite.com", "location of the cloud controller api")
	adminUser         = flag.String("admin-user", "admin", "uaa admin user")
	adminPassword     = flag.String("admin-password", "admin", "uaa admin password")
	appsDomain        = flag.String("apps-domain", "bosh-lite.com", "apps domain on cloud controller")
	skipSSLValidation = flag.Bool("skip-ssl-validation", true, "skip ssl validation")

	orgName   = flag.String("org-name", "stress-tests-org", "organization to use for stress tests")
	spaceName = flag.String("space-name", "stress-tests-space", "space to use for stress tests")

	cfLogsDirectory = flag.String("cf-logs-directory", "", "absolute path to directory in which to put cf logs")

	pusherID = flag.String("pusher-id", "", "unique identifier for this pusher instance")
)

func main() {
	cflager.AddFlags(flag.CommandLine)

	flag.Parse()

	logger, _ := cflager.New("worker-" + *pusherID)
	logger.Info("started")
	defer logger.Info("exited")

	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "logger", logger)

	poller := Poller{ctx: ctx, cancel: cancel}
	pusher := Pusher{ID: *pusherID, ctx: ctx, cancel: cancel}

	group := grouper.NewOrdered(os.Interrupt, grouper.Members{
		{"poller", poller},
		{"pusher", pusher},
	})

	monitor := ifrit.Invoke(group)
	err := <-monitor.Wait()
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}
}
