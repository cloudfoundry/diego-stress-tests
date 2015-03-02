package diego_test

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
	"github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type round struct {
	name string

	westley, max, princess, humperdink int
}

var _ = Describe("Launching and Running many CF applications", func() {
	BeforeEach(func() {
		runtime.GOMAXPROCS(runtime.NumCPU())
	})

	AfterEach(func() {
		runtime.GOMAXPROCS(1)
	})

	It("handles the load", func() {
		rounds := make([]round, 0, 40)
		for i := 0; i < 10; i++ {
			rounds = append(rounds, []round{
				{
					name:       fmt.Sprintf("round-%d-a", i),
					westley:    13,
					max:        3,
					princess:   1,
					humperdink: 3,
				},
				{
					name:       fmt.Sprintf("round-%d-b", i),
					westley:    13,
					max:        3,
					princess:   0,
					humperdink: 3,
				},
				{
					name:       fmt.Sprintf("round-%d-c", i),
					westley:    14,
					max:        3,
					princess:   1,
					humperdink: 2,
				},
				{
					name:       fmt.Sprintf("round-%d-d", i),
					westley:    14,
					max:        3,
					princess:   0,
					humperdink: 2,
				},
			}...)
		}

		startTime := time.Now()
		for _, round := range rounds {
			executeRound(round)
		}
		endTime := time.Now()

		file, err := os.Create(fmt.Sprintf("%s/total_time", stress_test_data_dir))
		Ω(err).ShouldNot(HaveOccurred())
		defer file.Close()

		file.WriteString(fmt.Sprintf("start: %s\nend: %s\nduration: %s\n", startTime, endTime, endTime.Sub(startTime)))
	})
})

func cf(outputFilePath string, timeout time.Duration, args ...string) int {
	sess := runner.Run("bash", "-c", fmt.Sprintf("CF_TRACE=true exec cf %s &>> %s", strings.Join(args, " "), outputFilePath))

	return sess.Wait(timeout).ExitCode()
}

func curlApp(appName, outputFile string) {
	startTime := time.Now()

	file, err := os.Create(outputFile)
	Ω(err).ShouldNot(HaveOccurred())

	var exitCode int

	defer func() {
		finalizeLogs(outputFile, appName, startTime, exitCode)
		file.Close()
	}()

	timer := time.NewTimer(CURL_RETRY_TIMEOUT).C
	for {
		exitCode = runner.Run("bash", "-c", fmt.Sprintf("exec curl -f %s.%s &>> %s", appName, os.Getenv("CF_APPS_DOMAIN"), outputFile)).Wait(CURL_TIMEOUT).ExitCode()
		if exitCode == 0 {
			return
		}

		select {
		case <-timer:
			return
		default:
		}
	}
}

func pushApp(appName, path, instances, memory, pushFilePath, logFilePath string) (*Session, *os.File) {
	startTime := time.Now()

	file, err := os.Create(pushFilePath)
	Ω(err).ShouldNot(HaveOccurred())

	var exitCode int

	defer func() {
		finalizeLogs(pushFilePath, appName, startTime, exitCode)
		file.Close()
	}()

	exitCode = cf(pushFilePath,
		CF_PUSH_TIMEOUT,
		"push", appName,
		"-p", path,
		"--no-start",
		"-b", "go_buildpack",
		"-i", instances,
		"-k", "1G",
		"-m", memory,
	)
	if exitCode != 0 {
		return nil, nil
	}

	logFile, err := os.Create(logFilePath)
	Ω(err).ShouldNot(HaveOccurred())

	logTailSession := runner.Run("bash", "-c", fmt.Sprintf("exec cf logs %s &>> %s", appName, logFilePath))

	exitCode = cf(pushFilePath, CF_CURL_TIMEOUT, "curl", fmt.Sprintf("v2/apps/`cf app %s --guid`", appName), "-X", "PUT", "-d", "'{\"diego\":true}'")
	if exitCode != 0 {
		return logTailSession, logFile
	}

	exitCode = cf(pushFilePath, CF_START_TIMEOUT, "start", appName)
	return logTailSession, logFile
}

func executeRound(r round) {
	westleyNames := generateNames("westley", r.westley)
	maxNames := generateNames("max", r.max)
	princessNames := generateNames("princess", r.princess)
	humperdinkNames := generateNames("humperdink", r.humperdink)

	err := os.MkdirAll(fmt.Sprintf("%s/%s", stress_test_data_dir, r.name), 0755)
	Ω(err).ShouldNot(HaveOccurred())

	wg := sync.WaitGroup{}
	for _, name := range westleyNames {
		name := name
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			defer wg.Done()

			pushAndCurl("westley", "1", "128M", r.name, name)
		}()
	}
	for _, name := range maxNames {
		name := name
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			defer wg.Done()

			pushAndCurl("max", "2", "512M", r.name, name)
		}()
	}
	for _, name := range princessNames {
		name := name
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			defer wg.Done()

			pushAndCurl("princess", "4", "1024M", r.name, name)
		}()
	}
	for _, name := range humperdinkNames {
		name := name
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			defer wg.Done()

			pushAndCurl("humperdink", "1", "128M", r.name, name)
		}()
	}
	wg.Wait()
}

func generateNames(prefix string, numNames int) []string {
	names := make([]string, numNames)
	for i := 0; i < numNames; i++ {
		guid, err := uuid.NewV4()
		Ω(err).ShouldNot(HaveOccurred())

		names[i] = fmt.Sprintf("%s-%s", prefix, guid)
	}

	return names
}

func finalizeLogs(outputFile, appName string, startTime time.Time, exitCode int) {
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	result := ""
	if exitCode == 0 {
		result = "SUCCESS"
	} else {
		result = "FAILURE"
	}

	Ω(runner.Run("bash", "-c", fmt.Sprintf("echo '%s: %v' &>> %s", result, duration, outputFile)).Wait()).Should(Exit(0))
}

func pushAndCurl(assetName, instancesArg, memoryArg, roundName, appName string) {
	logSession, logFile := pushApp(
		appName,
		"../assets/apps/"+assetName,
		instancesArg,
		memoryArg,
		fmt.Sprintf("%s/%s/push-%s", stress_test_data_dir, roundName, appName),
		fmt.Sprintf("%s/%s/log-%s", stress_test_data_dir, roundName, appName),
	)

	curlApp(appName, fmt.Sprintf("%s/%s/curl-%s", stress_test_data_dir, roundName, appName))

	time.Sleep(2 * time.Second)
	if logSession != nil {
		logSession.Kill()
	}
	if logFile != nil {
		logFile.Close()
	}
}
