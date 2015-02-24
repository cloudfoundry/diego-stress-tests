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
					name:       fmt.Sprintf("round-a-%d", i),
					westley:    13,
					max:        3,
					princess:   1,
					humperdink: 3,
				},
				{
					name:       fmt.Sprintf("round-b-%d", i),
					westley:    13,
					max:        3,
					princess:   0,
					humperdink: 3,
				},
				{
					name:       fmt.Sprintf("round-c-%d", i),
					westley:    14,
					max:        3,
					princess:   1,
					humperdink: 2,
				},
				{
					name:       fmt.Sprintf("round-d-%d", i),
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
	sess := runner.Run("bash", "-c", fmt.Sprintf("cf %s &>> %s", strings.Join(args, " "), outputFilePath))

	return sess.Wait(timeout).ExitCode()
}

func curlApp(appName, outputFile string) {
	startTime := time.Now()

	file, err := os.Create(outputFile)
	Ω(err).ShouldNot(HaveOccurred())

	var exitCode int

	defer func() {
		cf(outputFile, CF_LOGS_TIMEOUT, "logs", appName, "--recent")

		endTime := time.Now()
		duration := endTime.Sub(startTime)

		result := ""
		if exitCode == 0 {
			result = "SUCCESS"
		} else {
			result = "FAILURE"
		}

		Ω(runner.Run("bash", "-c", fmt.Sprintf("echo '%s: %v' &>> %s", result, duration, outputFile)).Wait()).Should(Exit(0))

		file.Close()
	}()

	exitCode = runner.Run("bash", "-c", fmt.Sprintf("curl -f %s.%s &>> %s", appName, os.Getenv("CF_APPS_DOMAIN"), outputFile)).Wait().ExitCode()
}

func pushApp(appName, path, instances, memory, outputFile string) {
	startTime := time.Now()

	file, err := os.Create(outputFile)
	Ω(err).ShouldNot(HaveOccurred())

	var exitCode int

	defer func() {
		cf(outputFile, CF_LOGS_TIMEOUT, "logs", appName, "--recent")

		endTime := time.Now()
		duration := endTime.Sub(startTime)

		result := ""
		if exitCode == 0 {
			result = "SUCCESS"
		} else {
			result = "FAILURE"
		}

		Ω(runner.Run("bash", "-c", fmt.Sprintf("echo '%s: %v' &>> %s", result, duration, outputFile)).Wait()).Should(Exit(0))

		file.Close()
	}()

	exitCode = cf(outputFile,
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
		return
	}

	exitCode = cf(outputFile, CF_SET_ENV_TIMEOUT, "set-env", appName, DIEGO_STAGE_BETA, "true")
	if exitCode != 0 {
		return
	}

	exitCode = cf(outputFile, CF_SET_ENV_TIMEOUT, "set-env", appName, DIEGO_RUN_BETA, "true")
	if exitCode != 0 {
		return
	}

	exitCode = cf(outputFile, CF_START_TIMEOUT, "start", appName)
	if exitCode != 0 {
		return
	}
}

func executeRound(r round) {
	westleyNames := make([]string, r.westley)
	for i := 0; i < r.westley; i++ {
		guid, err := uuid.NewV4()
		Ω(err).ShouldNot(HaveOccurred())

		westleyNames[i] = fmt.Sprintf("westley-%s", guid)
	}

	maxNames := make([]string, r.max)
	for i := 0; i < r.max; i++ {
		guid, err := uuid.NewV4()
		Ω(err).ShouldNot(HaveOccurred())

		maxNames[i] = fmt.Sprintf("max-%s", guid)
	}

	humperdinkNames := make([]string, r.humperdink)
	for i := 0; i < r.humperdink; i++ {
		guid, err := uuid.NewV4()
		Ω(err).ShouldNot(HaveOccurred())

		humperdinkNames[i] = fmt.Sprintf("humperdink-%s", guid)
	}

	princessNames := make([]string, r.princess)
	for i := 0; i < r.princess; i++ {
		guid, err := uuid.NewV4()
		Ω(err).ShouldNot(HaveOccurred())

		princessNames[i] = fmt.Sprintf("princess-%s", guid)
	}

	err := os.MkdirAll(fmt.Sprintf("%s/%s", stress_test_data_dir, r.name), 0755)
	Ω(err).ShouldNot(HaveOccurred())

	startTime := time.Now()
	wg := sync.WaitGroup{}
	for _, westley := range westleyNames {
		westley := westley
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			pushApp(westley, "../assets/apps/westley", "1", "128M", fmt.Sprintf("%s/%s/push-%s", stress_test_data_dir, r.name, westley))
			wg.Done()
		}()
	}

	for _, max := range maxNames {
		max := max
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			pushApp(max, "../assets/apps/max", "2", "512M", fmt.Sprintf("%s/%s/push-%s", stress_test_data_dir, r.name, max))
			wg.Done()
		}()
	}

	for _, princess := range princessNames {
		princess := princess
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			pushApp(princess, "../assets/apps/princess", "4", "1024M", fmt.Sprintf("%s/%s/push-%s", stress_test_data_dir, r.name, princess))
			wg.Done()
		}()
	}

	for _, humperdink := range humperdinkNames {
		humperdink := humperdink
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			pushApp(humperdink, "../assets/apps/humperdink", "1", "128M", fmt.Sprintf("%s/%s/push-%s", stress_test_data_dir, r.name, humperdink))
			wg.Done()
		}()
	}

	wg.Wait()
	endTime := time.Now()

	allPushesFile, err := os.Create(fmt.Sprintf("%s/%s/all_pushes", stress_test_data_dir, r.name))
	Ω(err).ShouldNot(HaveOccurred())

	defer allPushesFile.Close()
	allPushesFile.WriteString(fmt.Sprintf("start: %s\nend: %s\nduration: %s\n", startTime, endTime, endTime.Sub(startTime)))

	startTime = time.Now()
	wg = sync.WaitGroup{}
	for _, westley := range westleyNames {
		westley := westley
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			curlApp(westley, fmt.Sprintf("%s/%s/curl-%s", stress_test_data_dir, r.name, westley))
			wg.Done()
		}()
	}

	for _, max := range maxNames {
		max := max
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			curlApp(max, fmt.Sprintf("%s/%s/curl-%s", stress_test_data_dir, r.name, max))
			wg.Done()
		}()
	}

	for _, princess := range princessNames {
		princess := princess
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			curlApp(princess, fmt.Sprintf("%s/%s/curl-%s", stress_test_data_dir, r.name, princess))
			wg.Done()
		}()
	}

	for _, humperdink := range humperdinkNames {
		humperdink := humperdink
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			curlApp(humperdink, fmt.Sprintf("%s/%s/curl-%s", stress_test_data_dir, r.name, humperdink))
			wg.Done()
		}()
	}

	wg.Wait()
	endTime = time.Now()

	allCurlsFile, err := os.Create(fmt.Sprintf("%s/%s/all_curls", stress_test_data_dir, r.name))
	Ω(err).ShouldNot(HaveOccurred())

	defer allCurlsFile.Close()
	allCurlsFile.WriteString(fmt.Sprintf("start: %s\nend: %s\nduration: %s\n", startTime, endTime, endTime.Sub(startTime)))
}
