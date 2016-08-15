package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/net/context"

	"code.cloudfoundry.org/cflager"
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

func cf(ctx context.Context, args ...string) ([]byte, error) {
	// TODO setup output files for stdout, stderr, and trace logs
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}
	logger = logger.Session("cf", lager.Data{"args": args})
	cmd := exec.Command("cf", args...)
	c := make(chan error, 1)
	var output []byte = nil

	go func() {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			logger.Error("failed-starting-cf-command", err)
			c <- err
		}
		err = cmd.Start()
		if err != nil {
			logger.Error("failed-starting-cf-command", err)
			c <- err
		}

		output, err = ioutil.ReadAll(stdout)
		if err != nil {
			logger.Error("failed-starting-cf-command", err)
			c <- err
		}

		err = cmd.Wait()
		if err != nil {
			logger.Error("failed-running-cf-command", err)
			c <- err
		}
		c <- nil
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-c:
		if err != nil {
			return nil, err
		} else {
			return output, nil
		}
	}
}

func (a *cfApp) Push(ctx context.Context, assetDir string, timeout time.Duration) error {
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}
	logger = logger.Session("push", lager.Data{"app": a.appName})
	logger.Info("started")

	ctx, _ = context.WithTimeout(ctx, timeout)

	defer logger.Info("completed")

	_, err := cf(ctx, "push", a.appName, "-p", assetDir, "-f", a.manifestPath, "--no-start")
	if err != nil {
		return err
	}
	endpointToHit := fmt.Sprintf(AppRoutePattern, a.appName, a.domain)
	_, err = cf(ctx, "set-env", a.appName, "ENDPOINT_TO_HIT", endpointToHit)
	if err != nil {
		logger.Error("failed-to-set-env", err)
		return err
	}
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
	defer logger.Info("completed")

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
	logger.Debug("successful-response", lager.Data{"response": response})
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
