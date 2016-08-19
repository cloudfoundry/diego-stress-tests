package main

import (
	"io/ioutil"
	"os/exec"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/lager"
	"golang.org/x/net/context"
)

func cf(ctx context.Context, args ...string) ([]byte, error) {
	// Use logger from context object, or create logger if none provided
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
