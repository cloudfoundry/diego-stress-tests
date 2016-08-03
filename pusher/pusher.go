package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"

	"code.cloudfoundry.org/lager"
	"github.com/hashicorp/consul/api"
	"github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

type Pusher struct {
	ID        string
	ctx       context.Context
	cancel    context.CancelFunc
	directory string
	started   <-chan string
}

type Update struct {
	PusherId string
	Batch    int
}

func (p Pusher) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)

	for {
		logger := p.ctx.Value("logger").(lager.Logger)
		select {
		case signal := <-signals:
			logger.Info("signalled", lager.Data{"signal": signal})
			p.cancel()
		case orchestratorAddress := <-p.started:
			go p.startPusher(logger, orchestratorAddress)
		case <-p.ctx.Done():
		}
	}
	// p.cancel()
	// <-signals
	return nil
}

func (p Pusher) startPusher(logger lager.Logger, orchestratorAddress string) error {
	p.directory = *cfLogsDirectory

	consulClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		logger.Error("failed-building-consul-client", err)
		return err
	}

	ctx := p.setupOutputFiles(p.ctx, "cf.setup")
	defer ctx.Value("stdout").(io.Closer).Close()
	defer ctx.Value("stderr").(io.Closer).Close()

	err = os.Chdir(*appPath)
	if err != nil {
		logger.Error("failed-changing-into-app-dir", err, lager.Data{"app-path": appPath})
		return err
	}

	err = setupCFCLI(ctx)
	if err != nil {
		logger.Error("failed-setting-up-cf-cli", err)
		return err
	}

	kv := consulClient.KV()
	key := "diego-perf-pushers/" + p.ID
	pair, _, err := kv.Get(key, nil)
	if err != nil {
		logger.Error("failed-getting-key", err)
		return err
	}

	shouldFillUp := false
	if pair == nil {
		logger.Info("fresh-state")
		shouldFillUp = true
	} else if string(pair.Value) == "done" {
		logger.Info("already-filled-up")
	} else {
		shouldFillUp = true
		logger.Info("already-started")

		logger := logger.Session("cleaning-up")
		logger.Info("starting")
		// The previous push was interrupted; delete the space and start over
		err := p.cleanup(context.WithValue(p.ctx, "logger", logger))
		if err != nil {
			logger.Error("failed", err)
			return err
		}
		logger.Info("complete")
	}

	_, err = kv.Put(&api.KVPair{Key: key, Value: []byte("started")}, nil)
	if err != nil {
		logger.Error("failed-setting-key", err)
		return err
	}

	if shouldFillUp {
		logger = logger.Session("filling-up")
		logger.Info("starting")
		err := p.fillUp(context.WithValue(p.ctx, "logger", logger), *appPath, *concurrentPushes, *batches, orchestratorAddress)
		if err != nil {
			logger.Error("failed", err)
			return err
		}
		logger.Info("complete")
	}

	_, err = kv.Put(&api.KVPair{Key: key, Value: []byte("done")}, nil)
	if err != nil {
		logger.Error("failed-setting-key", err)
		return err
	}
	return nil
}

func (p Pusher) setupOutputFiles(ctx context.Context, prefix string) context.Context {
	logger := ctx.Value("logger").(lager.Logger)

	stdout := openFile(logger, path.Join(p.directory, prefix+".stdout.log"))
	ctx = context.WithValue(ctx, "stdout", stdout)

	stderr := openFile(logger, path.Join(p.directory, prefix+".stderr.log"))
	ctx = context.WithValue(ctx, "stderr", stderr)

	ctx = context.WithValue(ctx, "trace", path.Join(p.directory, prefix+".trace.log"))

	return ctx
}

type appDefinition struct {
	template       string
	appCount       int
	instancePerApp int
	appNamePrefix  string
}

func (a *appDefinition) GenerateManifest(domain, appGuid string) (string, error) {
	templ, err := template.ParseFiles(a.template)
	if err != nil {
		return "", err
	}

	filePath := filepath.Join("manifests", fmt.Sprintf("manifest-%s-%s.yml", a.appNamePrefix, appGuid))

	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}

	return filePath, templ.Execute(file, map[string]interface{}{
		"domain":       domain,
		"appName":      fmt.Sprintf("%s-%s", a.appNamePrefix, appGuid),
		"appInstances": a.instancePerApp,
	})
}

func (p Pusher) pushes(ctx context.Context, count int) error {
	logger := ctx.Value("logger").(lager.Logger)

	appTypes := []appDefinition{
		appDefinition{template: "manifest-light.yml.template", appCount: 9, instancePerApp: 1, appNamePrefix: "light"},
		appDefinition{template: "manifest-light.yml.template", appCount: 1, instancePerApp: 4, appNamePrefix: "light-group"},
		appDefinition{template: "manifest-medium.yml.template", appCount: 7, instancePerApp: 1, appNamePrefix: "medium"},
		appDefinition{template: "manifest-medium.yml.template", appCount: 1, instancePerApp: 2, appNamePrefix: "medium-group"},
		appDefinition{template: "manifest-heavy.yml.template", appCount: 1, instancePerApp: 1, appNamePrefix: "heavy"},
		appDefinition{template: "manifest-crashing.yml.template", appCount: 2, instancePerApp: 1, appNamePrefix: "crashing"},
	}

	totalAppCount := 0

	for appTypeIdx, _ := range appTypes {
		totalAppCount += appTypes[appTypeIdx].appCount
	}

	errChan := make(chan error, count*totalAppCount)
	wg := sync.WaitGroup{}

	rateLimiter := make(chan struct{}, count)
	for appTypeIdx, _ := range appTypes {
		for appIdx := 0; appIdx < appTypes[appTypeIdx].appCount; appIdx++ {
			appGuid := uuid.NewV4().String()
			logger := logger.Session("push", lager.Data{"appGuid": appGuid})

			// TODO: Should we combine these into a single file
			ctx := context.WithValue(ctx, "logger", logger)

			ctx = p.setupOutputFiles(ctx, "cf.push."+appGuid)
			defer ctx.Value("stdout").(io.Closer).Close()
			defer ctx.Value("stderr").(io.Closer).Close()

			// TODO: don't use package global variable
			manifestPath, err := appTypes[appTypeIdx].GenerateManifest(*appsDomain, appGuid)
			if err != nil {
				logger.Fatal("failed-generating-app-manifest", err)
			}

			wg.Add(1)

			go func() {
				rateLimiter <- struct{}{}
				logger.Info("starting")
				defer func() {
					logger.Info("complete")
					<-rateLimiter
					wg.Done()
				}()

				var err error
				for tries := *pushRetries; tries > 0; tries-- {
					logger.Info("attempting-push", lager.Data{"manifest": manifestPath})
					err = push(ctx, "-f", manifestPath)
					if err != nil {
						logger.Error("failed-pushing-app", nil, lager.Data{"attempt": *pushRetries - tries + 1})
						continue
					}

					logger.Info("successful-push")
					break
				}

				if err != nil {
					logger.Error("giving-up-pushing-app", nil)
					errChan <- err
					p.cancel()
				}
			}()
		}
	}

	wg.Wait()

	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

func (p Pusher) fillUp(ctx context.Context, appPath string, concurrentPushes int, batches int, orchestratorAddress string) error {
	logger := ctx.Value("logger").(lager.Logger)
	errChan := make(chan error, batches)

	for i := 0; i < batches; i++ {
		logger := logger.Session("batch", lager.Data{"batch": i + 1})
		logger.Info("starting")

		err := p.pushes(context.WithValue(ctx, "logger", logger), concurrentPushes*batches)
		if err != nil {
			errChan <- err
		}

		// Post update to orchestrator
		url := "http://" + orchestratorAddress + "/v1/diego-perf/update"
		logger.Info("posting-update", lager.Data{"orchestrator-url": url})
		payload := Update{PusherId: p.ID, Batch: i + 1}
		out, err := json.Marshal(&payload)
		if err != nil {
			logger.Fatal("failed-to-marshal", err)
		}
		jsonBuffer := bytes.NewBuffer(out)
		logger.Info("posting-update", lager.Data{"payload": payload})
		resp, err := http.Post(url, "application/json", jsonBuffer)
		if err != nil {
			logger.Fatal("failed-to-post-update", err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			err := fmt.Errorf("status code: %d", resp.StatusCode)
			logger.Fatal("failed-post-update-with-non-200", err)
		}

		logger.Info("complete")
	}

	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

func (p Pusher) cleanup(ctx context.Context) error {
	ctx = p.setupOutputFiles(ctx, "cf.cleanup")
	defer ctx.Value("stdout").(io.Closer).Close()
	defer ctx.Value("stderr").(io.Closer).Close()

	logger := ctx.Value("logger").(lager.Logger).Session("cleanup")
	ctx = context.WithValue(ctx, "logger", logger)

	err := cf(ctx, CFDefaultTimeout, "delete-space", "-f", *spaceName)
	if err != nil {
		return err
	}

	err = cf(ctx, CFDefaultTimeout, "create-space", *spaceName, "-o", *orgName)
	if err != nil {
		return err
	}

	err = cf(ctx, CFDefaultTimeout, "target", "-o", *orgName, "-s", *spaceName)
	if err != nil {
		return err
	}

	return nil
}
