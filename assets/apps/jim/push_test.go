package main_test

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Push", func() {
	It("pushes apps", func() {
		pushes := func(from, to int) {
			wg := sync.WaitGroup{}
			for i := from; i <= to; i++ {
				i := i
				err := generateManifest("manifest.yml.template", i)
				Expect(err).NotTo(HaveOccurred())

				wg.Add(1)

				go func() {
					defer GinkgoRecover()

					exitCode := push("-f", fmt.Sprintf("manifests/manifest-%d.yml", i))
					Expect(exitCode).To(Equal(0))
					wg.Done()
				}()
			}
			wg.Wait()
		}

		fmt.Fprintln(GinkgoWriter, "Starting batch 1")
		pushes(1, 3)

		fmt.Fprintln(GinkgoWriter, "Starting batch 2")
		pushes(4, 7)

		fmt.Fprintln(GinkgoWriter, "Starting batch 3")
		pushes(8, 10)
	})
})

func cf(timeout time.Duration, args ...string) int {
	cmd := fmt.Sprintf("cf %s", strings.Join(args, " "))
	sess := runner.Run("bash", "-c", cmd)
	return sess.Wait(timeout).ExitCode()
}

func push(args ...string) int {
	return cf(CFPushTimeout, append([]string{"push"}, args...)...)
}

func generateManifest(templatePath string, index int) error {
	t, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}

	f, err := os.Create(fmt.Sprintf("manifests/manifest-%d.yml", index))
	if err != nil {
		return err
	}
	err = t.Execute(f, map[string]string{
		"domain":       config.AppsDomain,
		"lightName":    fmt.Sprintf("light-%05d", index),
		"mediumName":   fmt.Sprintf("medium-%05d", index),
		"heavyName":    fmt.Sprintf("heavy-%05d", index),
		"crashingName": fmt.Sprintf("crashing-%05d", index),
	})
	if err != nil {
		return err
	}
	return nil
}
