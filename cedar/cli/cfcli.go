package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"github.com/glycerine/rbuf"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/lager"
	"golang.org/x/net/context"
)

//go:generate counterfeiter -o fakes/fake_cfclient.go . CFClient
type CFClient interface {
	Cf(logger lager.Logger, ctx context.Context, timeout time.Duration, args ...string) ([]byte, error)
	Cleanup(ctx context.Context) error
	Pool() chan string
}

type CFPooledClient struct {
	poolSize int
	pool     chan string
	homeDir  string
}

func NewCfClient(ctx context.Context, poolSize int) CFClient {
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}
	logger = logger.Session("cf")
	user, err := user.Current()
	if err != nil {
		logger.Error("get-home-dir-failed", err)
	}
	homeDir := user.HomeDir

	if _, err = os.Stat(filepath.Join(homeDir, ".cf")); os.IsNotExist(err) {
		logger.Error("cf-dir-unavailable", err)
		panic("cf-dir-unavailable")
	}

	pool := make(chan string, poolSize)
	for i := 0; i < poolSize; i++ {
		cfDir, err := ioutil.TempDir("", "cfhome")
		if err != nil {
			logger.Error("init-temp-cf-dir-failed", err)
		}

		cmd := exec.Command("cp", "-r", filepath.Join(homeDir, ".cf"), filepath.Join(cfDir, ".cf"))
		err = cmd.Run()
		if err != nil {
			logger.Error("copy-cf-config-failed", err)
		}
		pool <- cfDir
	}

	return &CFPooledClient{
		homeDir:  homeDir,
		pool:     pool,
		poolSize: poolSize,
	}
}

func (cfcli *CFPooledClient) Pool() chan string {
	return cfcli.pool
}

func (cfcli *CFPooledClient) Cf(logger lager.Logger, ctx context.Context, timeout time.Duration, args ...string) ([]byte, error) {
	logger = logger.Session("cf", lager.Data{"args": args})

	cfDir := <-cfcli.pool
	os.Setenv("CF_HOME", cfDir)
	defer func() { cfcli.pool <- cfDir }()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	cmd := exec.CommandContext(ctx, "cf", args...)
	cmd.Env = append(os.Environ(), "GOMAXPROCS=4")

	buf := rbuf.NewFixedSizeRingBuf(1024)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("failed-getting-command-stdout", err)
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Error("failed-getting-command-stderr", err)
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		logger.Error("failed-starting-cf-command", err)
		return nil, err
	}

	_, err = buf.ReadFrom(io.MultiReader(stdout, stderr))
	if err != nil {
		logger.Error("failed-reading-command-output", err)
		// we shouldn't exit yet, until we wait for the subprocess to exit
		cancel()
	}

	err = cmd.Wait()
	if err != nil {
		logger.Error("failed-running-cf-command", err, lager.Data{"stdout": string(buf.Bytes())})
		return nil, err
	}
	return buf.Bytes(), nil
}

func (cfcli *CFPooledClient) Cleanup(ctx context.Context) error {
	logger, ok := ctx.Value("logger").(lager.Logger)
	if !ok {
		logger, _ = cflager.New("cedar")
	}
	logger = logger.Session("cf-cleanup")
	logger.Info("started", lager.Data{"tmp-dir-size": len(cfcli.pool)})
	defer logger.Info("completed")

	if len(cfcli.pool) != cfcli.poolSize {
		return fmt.Errorf("pool-size-mismatch")
	}

	for tmpDir := range cfcli.pool {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			logger.Error("failed-to-remove-tmpdir", err, lager.Data{"dir": tmpDir})
		}
		if len(cfcli.pool) == 0 {
			close(cfcli.pool)
		}
	}

	return nil
}
