package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/context"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/lager"
)

const (
	AppRoutePattern = "http://%s.%s"
)

type CfApp interface {
	AppName() string
	Url() string
	Push(context context.Context, payload string, timeout time.Duration) error
	Start(context context.Context, timeout time.Duration) error
	Guid(context context.Context, timeout time.Duration) (string, error)
}

type cfApp struct {
	appName        string
	appRoute       url.URL
	attemptedCurls int
	failedCurls    int
	domain         string
	maxFailedCurls int
	manifestPath   string
}

func NewCfApp(appName string, domain string, maxFailedCurls int, manifestPath string) (CfApp, error) {
	rawUrl := fmt.Sprintf(AppRoutePattern, appName, domain)
	appUrl, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}
	return &cfApp{
		appName:        appName,
		appRoute:       *appUrl,
		domain:         domain,
		maxFailedCurls: maxFailedCurls,
		manifestPath:   manifestPath,
	}, nil
}

func (a *cfApp) AppName() string {
	return a.appName
}

func (a *cfApp) Url() string {
	return a.appRoute.String()
}

func (a *cfApp) Push(ctx context.Context, assetDir string, timeout time.Duration) error {
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}
	logger = logger.Session("push", lager.Data{"app": a.appName})
	logger.Info("started")

	ctx, _ = context.WithTimeout(ctx, timeout)

	_, err := cf(ctx, "push", a.appName, "-p", assetDir, "-f", a.manifestPath, "--no-start")
	if err != nil {
		logger.Error("failed-to-push", err)
		return err
	}
	endpointToHit := fmt.Sprintf(AppRoutePattern, a.appName, a.domain)
	_, err = cf(ctx, "set-env", a.appName, "ENDPOINT_TO_HIT", endpointToHit)
	if err != nil {
		logger.Error("failed-to-set-env", err)
		return err
	}
	logger.Info("completed")
	logger.Debug("successful-set-env", lager.Data{"ENDPOINT_TO_HIT": endpointToHit})
	return nil
}

func (a *cfApp) Start(ctx context.Context, timeout time.Duration) error {
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}
	logger = logger.Session("start", lager.Data{"app": a.appName})
	logger.Info("started")

	ctx, _ = context.WithTimeout(ctx, timeout)

	_, err := cf(ctx, "start", a.appName)
	if err != nil {
		logger.Error("failed-to-start", err)
		return err
	}

	response, err := a.Curl("")
	if err != nil {
		logger.Error("failed-curling-app", err)
		return err
	}
	logger.Info("completed")
	logger.Debug("successful-response-starting", lager.Data{"response": response})
	return nil
}

func (a *cfApp) Guid(ctx context.Context, timeout time.Duration) (string, error) {
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}
	logger = logger.Session("guid", lager.Data{"app": a.appName})
	logger.Info("started")
	defer logger.Info("completed")

	ctx, _ = context.WithTimeout(ctx, timeout)
	output, err := cf(ctx, "app", "--guid", a.appName)

	if err != nil {
		logger.Error("failed-to-get-guid", err)
		return "", err
	}
	return strings.Trim(string(output), "\n"), nil
}

func (a *cfApp) Curl(endpoint string) (string, error) {
	endpointUrl := a.appRoute
	endpointUrl.Path = endpoint

	url := endpointUrl.String()

	statusCode, body, err := curl(url)
	if err != nil {
		return "", err
	}

	a.attemptedCurls++

	switch {
	case statusCode == 200:
		return string(body), nil

	case a.shouldRetryRequest(statusCode):
		fmt.Println("RETRYING CURL", newCurlErr(url, statusCode, body).Error())
		a.failedCurls++
		time.Sleep(2 * time.Second)
		return a.Curl(endpoint)

	default:
		err := newCurlErr(url, statusCode, body)
		fmt.Println("FAILED CURL", err.Error())
		a.failedCurls++
		return "", err
	}
}

func (a cfApp) shouldRetryRequest(statusCode int) bool {
	if statusCode == 503 || statusCode == 404 {
		return a.failedCurls < a.maxFailedCurls
	}
	return false
}
