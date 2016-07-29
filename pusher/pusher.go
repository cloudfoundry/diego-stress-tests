package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sync"

	"code.cloudfoundry.org/lager"
	"github.com/hashicorp/consul/api"
	"github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

type Pusher struct {
	ID string

	ctx    context.Context
	cancel context.CancelFunc

	directory           string
	orchestratorAddress *string
}

type Update struct {
	PusherId string
	Batch    int
}

func (p Pusher) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	logger := p.ctx.Value("logger").(lager.Logger)
	go func() {
		select {
		case <-signals:
			p.cancel()
		case <-p.ctx.Done():
		}
	}()

	p.directory = *cfLogsDirectory

	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		logger.Error("failed-building-consul-client", err)
		return err
	}

	kv := client.KV()
	key := "diego-perf-pushers/" + p.ID

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
		// deleted things
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
		err := p.fillUp(context.WithValue(p.ctx, "logger", logger), *appPath, *batchSize, *batches)
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

	close(ready)

	p.cancel()
	<-signals
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

func (p Pusher) pushes(ctx context.Context, count int) error {
	logger := ctx.Value("logger").(lager.Logger)
	errChan := make(chan error, count)

	wg := sync.WaitGroup{}
	for i := 0; i < count; i++ {
		guid := uuid.NewV4().String()
		logger := logger.Session("push", lager.Data{"guid": guid})
		ctx := context.WithValue(ctx, "logger", logger)

		ctx = p.setupOutputFiles(ctx, "cf.push."+guid)
		defer ctx.Value("stdout").(io.Closer).Close()
		defer ctx.Value("stderr").(io.Closer).Close()

		err := generateManifest(*appsDomain, "manifest.yml.template", guid)
		if err != nil {
			logger.Error("failed-generating-app-manifest", err)
			os.Exit(1)
		}

		wg.Add(1)

		go func() {
			logger.Info("starting")
			defer func() {
				logger.Info("complete")
				wg.Done()
			}()

			for tries := *pushRetries; tries > 0; tries-- {
				logger.Info("attempting-push")
				err := push(ctx, "-f", fmt.Sprintf("manifests/manifest-%s.yml", guid))
				if err != nil {
					logger.Error("failed-pushing-app", nil, lager.Data{"attempt": *pushRetries - tries + 1})
					continue
				}
				logger.Info("successful-push")
				return
			}
			logger.Error("giving-up-pushing-app", nil)
			errChan <- err
			p.cancel()
		}()
	}
	wg.Wait()

	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

func (p Pusher) fillUp(ctx context.Context, appPath string, batchSize int, batches int) error {
	logger := ctx.Value("logger").(lager.Logger)
	errChan := make(chan error, batches)

	for i := 0; i < batches; i++ {
		logger := logger.Session("batch", lager.Data{"batch": i + 1})
		logger.Info("starting")
		err := p.pushes(context.WithValue(ctx, "logger", logger), batchSize)
		errChan <- err
		// Post update to orchestrator
		host := *p.orchestratorAddress
		url := "http://" + host + "/v1/diego-perf/update"

		payload := Update{PusherId: p.ID, Batch: i + 1}
		jsonBuffer := &bytes.Buffer{}
		json.NewEncoder(jsonBuffer).Encode(&payload)
		logger.Info("posting-update")
		resp, err := http.Post(url, "application/json", jsonBuffer)
		if err != nil {
			logger.Fatal("failed-to-post-update", err)
			return err
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			err := fmt.Errorf("status code: %d", resp.StatusCode)
			logger.Fatal("failed-post-update-with-non-200", err)
			return err
		}

		logger.Info("complete")
	}

	select {
	case err := <-errChan:
		return err
	}

	return nil
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
