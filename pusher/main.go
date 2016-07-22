package main

import (
	"flag"
	"os"
	"time"

	"golang.org/x/net/context"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/lager"

	"github.com/hashicorp/consul/api"
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
)

func main() {
	cflager.AddFlags(flag.CommandLine)

	flag.Parse()

	logger, reconfigurableSink := cflager.New("worker")
	reconfigurableSink.SetMinLevel(lager.DEBUG)
	logger.Info("starting")
	defer logger.Info("complete")
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "logger", logger)

	pusher := Pusher{ctx: ctx, cancel: cancel}
	poller := Poller{ctx: ctx, cancel: cancel}

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

type Poller struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func (p Poller) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	logger := p.ctx.Value("logger").(lager.Logger)
	logger = logger.Session("poller")

	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		logger.Error("failed-building-consul-client", err)
		return err
	}

	kv := client.KV()
	ticker := time.NewTicker(time.Second)

	logger.Info("starting")

	for {
		<-ticker.C

		key := "diego_perf_pusher_start"
		logger.Debug("polling-for-key", lager.Data{"key": key})
		pair, _, err := kv.Get(key, nil)
		if err != nil {
			logger.Error("failed-polling-for-key", err)
			return err
		}

		if pair != nil {
			logger.Info("found-key", lager.Data{"key": key, "value": pair})
			close(ready)
			break
		}
	}

	logger.Info("complete")

	<-signals
	return nil
}
