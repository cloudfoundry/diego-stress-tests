package cli_test

import (
	"time"

	. "code.cloudfoundry.org/diego-stress-tests/cedar/cli"
	"golang.org/x/net/context"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega/gbytes"

	. "github.com/onsi/gomega"
)

var _ = Describe("Cli", func() {
	var ctx context.Context
	var client CFClient
	var pool chan string
	var cfDir1, cfDir2 string

	BeforeEach(func() {
		ctx, _ = context.WithCancel(
			context.WithValue(context.Background(),
				"logger",
				fakeLogger,
			),
		)
		client = NewCfClient(ctx, 2)
		pool = client.Pool()
	})

	AfterEach(func() {
		client.Cleanup(ctx)
		Expect(len(client.Pool())).To(Equal(0))
		Expect(cfDir1).ToNot(BeADirectory())
		Expect(cfDir2).ToNot(BeADirectory())
	})

	Context("when cli is created", func() {
		BeforeEach(func() {
			Expect(len(client.Pool())).To(Equal(2))
			cfDir1 = <-pool
			cfDir2 = <-pool
		})

		AfterEach(func() {
			pool <- cfDir1
			pool <- cfDir2
		})

		It("creates pool of cf home directories", func() {
			Expect(cfDir1).ShouldNot(Equal(cfDir2))
			Expect(cfDir1).To(ContainSubstring("cfhome"))
			Expect(cfDir1).To(BeADirectory())
			Expect(cfDir2).To(BeADirectory())
		})
	})

	Context("When an error is returned by the cli", func() {
		It("logs to show the error from cli when pushing without arguments", func() {
			_, err := client.Cf(fakeLogger, ctx, 30*time.Second, "push")
			Expect(fakeLogger).To(gbytes.Say("failed-running-cf-command"))
			Expect(fakeLogger).To(gbytes.Say("\"log_level\":2"))
			Expect(fakeLogger).To(gbytes.Say("FAILED"))
			Expect(err).To(HaveOccurred())
		})

		It("logs to show the error from cli when start without arguments", func() {
			_, err := client.Cf(fakeLogger, ctx, 30*time.Second, "start")
			Expect(fakeLogger).To(gbytes.Say("failed-running-cf-command"))
			Expect(fakeLogger).To(gbytes.Say("\"log_level\":2"))
			Expect(fakeLogger).To(gbytes.Say("FAILED"))
			Expect(err).To(HaveOccurred())
		})
	})
})
