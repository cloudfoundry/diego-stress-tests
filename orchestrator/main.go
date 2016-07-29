package main

import (
	"flag"
	"os"

	"github.com/hashicorp/consul/api"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"

	"code.cloudfoundry.org/cflager"
)

var listenAddress = flag.String(
	"listenAddress",
	"",
	"The host:port that the server is bound to.",
)

var client *api.Client
var err error

var (
	expectedNumPushers = flag.Int("expectedNumPushers", -1, "maximum number of tries for a single push")
)

func main() {
	cflager.AddFlags(flag.CommandLine)
	flag.Parse()

	logger, _ := cflager.New("orchestrator")
	logger.Info("started")
	defer logger.Info("exited")

	handler := New(logger)

	var server ifrit.Runner
	server = http_server.New(*listenAddress, handler)
	stopPushersPoller := NewStopPushersPoller(logger)

	group := grouper.NewParallel(os.Interrupt, grouper.Members{
		{"server", server},
		{"stopPushers", stopPushersPoller},
	})

	client, err = api.NewClient(api.DefaultConfig())

	monitor := ifrit.Invoke(group)
	err = <-monitor.Wait()
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}
}
