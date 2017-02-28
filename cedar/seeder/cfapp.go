package seeder

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/context"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/diego-stress-tests/cedar/cli"

	"code.cloudfoundry.org/lager"
)

const (
	AppRoutePattern = "%s://%s.%s"
)

//go:generate counterfeiter -o fakes/fake_cfapp.go . CfApp
type CfApp interface {
	AppName() string
	AppURL() string
	Push(logger lager.Logger, ctx context.Context, client cli.CFClient, payload string, timeout time.Duration) error
	Start(logger lager.Logger, ctx context.Context, client cli.CFClient, skipVerifyCertificate bool, timeout time.Duration) error
	Guid(logger lager.Logger, ctx context.Context, client cli.CFClient, timeout time.Duration) (string, error)
}

type CfApplication struct {
	appName        string
	appRoute       url.URL
	attemptedCurls int
	failedCurls    int
	domain         string
	useSSL         bool
	maxFailedCurls int
	manifestPath   string
}

func NewCfApp(appName string, domain string, useSSL bool, maxFailedCurls int, manifestPath string) (CfApp, error) {
	protocol := "http"
	if useSSL {
		protocol = "https"
	}

	rawUrl := fmt.Sprintf(AppRoutePattern, protocol, appName, domain)

	appUrl, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}
	return &CfApplication{
		appName:        appName,
		appRoute:       *appUrl,
		domain:         domain,
		useSSL:         useSSL,
		maxFailedCurls: maxFailedCurls,
		manifestPath:   manifestPath,
	}, nil
}

func (a *CfApplication) AppName() string {
	return a.appName
}

func (a *CfApplication) AppURL() string {
	return a.appRoute.String()
}

func (a *CfApplication) Push(logger lager.Logger, ctx context.Context, cli cli.CFClient, assetDir string, timeout time.Duration) error {
	logger = logger.Session("push", lager.Data{"app": a.appName})
	logger.Info("started")

	_, err := cli.Cf(logger, ctx, timeout, "push", a.appName, "-p", assetDir, "-f", a.manifestPath, "--no-start")
	if err != nil {
		logger.Error("failed-to-push", err)
		return err
	}

	endpointToHit := a.AppURL()
	_, err = cli.Cf(logger, ctx, timeout, "set-env", a.appName, "ENDPOINT_TO_HIT", endpointToHit)
	if err != nil {
		logger.Error("failed-to-set-env", err)
		return err
	}
	logger.Info("completed")
	logger.Debug("successful-set-env", lager.Data{"ENDPOINT_TO_HIT": endpointToHit})
	return nil
}

func (a *CfApplication) Start(logger lager.Logger, ctx context.Context, cli cli.CFClient, skipVerifyCertificate bool, timeout time.Duration) error {
	logger = logger.Session("start", lager.Data{"app": a.appName})
	logger.Info("started")

	_, err := cli.Cf(logger, ctx, timeout, "start", a.appName)
	if err != nil {
		logger.Error("failed-to-start", err)
		return err
	}
	response, err := a.curl(ctx, skipVerifyCertificate)
	if err != nil {
		logger.Error("failed-curling-app", err)
		return err
	}
	logger.Info("completed")
	logger.Debug("successful-response-starting", lager.Data{"response": response})
	return nil
}

func (a *CfApplication) Guid(logger lager.Logger, ctx context.Context, cli cli.CFClient, timeout time.Duration) (string, error) {
	logger = logger.Session("guid", lager.Data{"app": a.appName})
	logger.Info("started")
	defer logger.Info("completed")

	output, err := cli.Cf(logger, ctx, timeout, "app", "--guid", a.appName)

	if err != nil {
		logger.Error("failed-to-get-guid", err)
		return "", err
	}
	return strings.Trim(string(output), "\n"), nil
}

func (a *CfApplication) SetUrl(appUrl string) error {
	appRoute, err := url.Parse(appUrl)
	if err != nil {
		return err
	}
	a.appRoute = *appRoute
	return nil
}

func (a *CfApplication) curl(ctx context.Context, skipVerifyCertificate bool) (string, error) {
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}

	logger = logger.Session("curl", lager.Data{"app": a.appName})
	logger.Info("started")
	defer logger.Info("completed")

	endpointUrl := a.appRoute
	endpointUrl.Path = ""

	url := endpointUrl.String()

	client := http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerifyCertificate},
	}}
	resp, err := client.Get(url)

	if err != nil {
		logger.Error("failed-to-curl", err)
		return "", err
	}

	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("failed-to-curl", err)
		return "", err
	}

	statusCode, body := resp.StatusCode, string(bytes)

	a.attemptedCurls++
	switch {
	case statusCode == 200:
		return string(body), nil

	case a.shouldRetryRequest(statusCode):
		a.failedCurls++
		logger.Error("retrying-curl", err, lager.Data{"url": url, "status-code": statusCode, "body": body, "retry": a.failedCurls})
		time.Sleep(2 * time.Second)
		return a.curl(ctx, skipVerifyCertificate)

	default:
		logger.Error("failed-to-curl", err, lager.Data{"url": url, "status-code": statusCode, "body": body})
		a.failedCurls++
		return "", errors.New("failed to curl app url")
	}
}

func (a CfApplication) shouldRetryRequest(statusCode int) bool {
	if statusCode == 503 || statusCode == 404 {
		return a.failedCurls < a.maxFailedCurls
	}
	return false
}
