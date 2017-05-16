package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"code.cloudfoundry.org/bbs"
	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/diego-stress-tests/drd/client"
	"code.cloudfoundry.org/diego-stress-tests/drd/diagnosis"
	"code.cloudfoundry.org/diego-stress-tests/drd/parser"
	"code.cloudfoundry.org/lager"
)

const (
	clientSessionCacheSize int = 0
	maxIdleConnsPerHost    int = 0
)

var (
	cedarInput        = flag.String("cedarInput", "", "cedar input file")
	outputFile        = flag.String("outputFile", "", "DrD output file")
	bbsSkipCertVerify = flag.Bool("bbsSkipCertVerify", false, "when set to true, skips all SSL/TLS certificate verification [environment variable equivalent: BBS_SKIP_CERT_VERIFY]")
	bbsURL            = flag.String("bbsURL", "", "URL of BBS server to target [environment variable equivalent: BBS_URL]")
	bbsCertFile       = flag.String("bbsCertFile", "", "path to the TLS client certificate to use during mutual-auth TLS [environment variable equivalent: BBS_CERT_FILE]")
	bbsKeyFile        = flag.String("bbsKeyFile", "", "path to the TLS client private key file to use during mutual-auth TLS [environment variable equivalent: BBS_KEY_FILE]")
	bbsCACertFile     = flag.String("bbsCACertFile", "", "path the Certificate Authority (CA) file to use when verifying TLS keypairs [environment variable equivalent: BBS_CA_CERT_FILE]")

	bbsClient bbs.Client
	err       error
)

func main() {
	cflager.AddFlags(flag.CommandLine)
	flag.Parse()

	logger := lager.NewLogger("drd")

	if !strings.HasPrefix(*bbsURL, "https") {
		bbsClient = bbs.NewClient(*bbsURL)
	} else {
		if *bbsSkipCertVerify {
			bbsClient, err = bbs.NewSecureSkipVerifyClient(
				*bbsURL,
				*bbsCertFile,
				*bbsKeyFile,
				clientSessionCacheSize,
				maxIdleConnsPerHost,
			)
		} else {
			bbsClient, err = bbs.NewSecureClient(
				*bbsURL,
				*bbsCACertFile,
				*bbsCertFile,
				*bbsKeyFile,
				clientSessionCacheSize,
				maxIdleConnsPerHost,
			)
		}
	}

	desiredLRPs, err := client.DesiredLRPs(logger, bbsClient, "")
	if err != nil {
		panic(err)
	}

	apps, err := parser.ParseAppFile(logger, *cedarInput)
	if err != nil {
		panic(err)
	}

	aggregateSummary := diagnosis.Summary{}
	for _, app := range apps {
		logger := logger.Session("diagnose-app")
		desiredLRP := diagnosis.DiscoverProcessGuid(app, desiredLRPs)
		if desiredLRP == nil {
			logger.Error("missing-app-info", fmt.Errorf("app not found"))
			continue
		}

		actualLRPs, err := client.ActualLRPGroupsForGuid(logger, bbsClient, desiredLRP.ProcessGuid)
		if err != nil {
			logger.Error("failed-reading-actual-lrp", err)
		}
		summary := diagnosis.DiagnoseApp(app, *desiredLRP, actualLRPs)
		aggregateSummary = diagnosis.JoinSummaries(aggregateSummary, summary)
	}

	summaryBytes, err := json.Marshal(aggregateSummary)
	if err != nil {
		panic(err)
	}
	fmt.Println(summaryBytes)
}
