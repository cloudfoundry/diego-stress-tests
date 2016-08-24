package cli_test

import (
	. "code.cloudfoundry.org/diego-stress-tests/cedar/cli"
	"golang.org/x/net/context"

	. "github.com/onsi/ginkgo"

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
		err := client.Cleanup(ctx)
		Expect(err).NotTo(HaveOccurred())
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
})
