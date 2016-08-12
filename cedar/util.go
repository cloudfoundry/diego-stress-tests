package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"code.cloudfoundry.org/lager"
)

const (
	AppRoutePattern = "http://%s.%s"
)

type cfApp struct {
	appName        string
	appRoute       url.URL
	attemptedCurls int
	failedCurls    int
	domain         string
	maxFailedCurls int
	manifestPath   string
}

func newCfApp(logger lager.Logger, appName string, domain string, maxFailedCurls int, manifestPath string) *cfApp {
	logger = logger.Session("creating-new-cf-app", lager.Data{"app": appName})
	logger.Debug("started")
	defer logger.Debug("completed")

	rawUrl := fmt.Sprintf(AppRoutePattern, appName, domain)
	appUrl, err := url.Parse(rawUrl)
	if err != nil {
		logger.Error("failed-parsing-url", err, lager.Data{"rawUrl": rawUrl})
		os.Exit(1)
	}
	return &cfApp{
		appName:        appName,
		appRoute:       *appUrl,
		domain:         domain,
		maxFailedCurls: maxFailedCurls,
		manifestPath:   manifestPath,
	}
}

func cf(logger lager.Logger, args ...string) error {
	// TODO timeout through context?
	// TODO setup output files for stdout, stderr, and trace logs
	logger = logger.Session("cf", lager.Data{"args": args})
	cmd := exec.Command("cf", args...)
	err := cmd.Start()
	if err != nil {
		logger.Error("failed-starting-cf-command", err)
		os.Exit(1)
	}

	errChan := make(chan error)
	go func() {
		errChan <- cmd.Wait()
	}()

	select {
	case err := <-errChan:
		if err != nil {
			logger.Error("failed-running-cf-command", err)
			return err
		}
	}
	return nil
}

func (a *cfApp) Push(logger lager.Logger, assetDir string) error {
	// push dummy app
	logger = logger.Session("push", lager.Data{"app": a.appName})
	logger.Info("started")
	defer logger.Info("completed")

	err := cf(logger, "push", a.appName, "-p", assetDir, "-f", a.manifestPath, "--no-start")
	if err != nil {
		return err
	}
	endpointToHit := fmt.Sprintf(AppRoutePattern, a.appName, a.domain)
	err = cf(logger, "set-env", a.appName, "ENDPOINT_TO_HIT", endpointToHit)
	if err != nil {
		logger.Error("failed-to-set-env", err)
		return err
	}
	logger.Debug("successful-set-env", lager.Data{"ENDPOINT_TO_HIT": endpointToHit})
	return nil
}

func (a *cfApp) Start(logger lager.Logger) error {
	logger = logger.Session("start", lager.Data{"app": a.appName})
	logger.Info("started")
	defer logger.Info("completed")

	err := cf(logger, "start", a.appName)
	if err != nil {
		logger.Error("failed-to-start", err)
		return err
	}

	response, err := a.Curl("")
	if err != nil {
		logger.Error("failed-curling-app", err)
		return err
	}
	logger.Debug("successful-response", lager.Data{"response": response})
	return nil
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

func (a *cfApp) shouldRetryRequest(statusCode int) bool {
	if statusCode == 503 || statusCode == 404 {
		return a.failedCurls < a.maxFailedCurls
	}
	return false
}

func curl(url string) (statusCode int, body string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, "", err
	}

	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}
	return resp.StatusCode, string(bytes), nil
}

func newCurlErr(url string, statusCode int, body string) error {
	return fmt.Errorf("Endpoint: %s, Status Code: %d, Body: %s", url, statusCode, body)
}
