package cli_test

import (
	"context"
	"errors"

	"code.cloudfoundry.org/diego-stress-tests/cedar/cli"
	"code.cloudfoundry.org/diego-stress-tests/cedar/cli/fakes"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SharedDomain", func() {
	var (
		ctx         context.Context
		cfCli       *fakes.FakeCFClient
		result      string
		returnedErr error
		testLogger  *lagertest.TestLogger
	)

	BeforeEach(func() {
		result = ""
		returnedErr = nil
		testLogger = lagertest.NewTestLogger("cfcli")
		ctx = context.WithValue(context.Background(), "logger", testLogger)
		cfCli = &fakes.FakeCFClient{}
	})

	JustBeforeEach(func() {
		cfCli.CfReturns([]byte(result), returnedErr)
	})

	Context("when cf curl returns a json", func() {
		BeforeEach(func() {
			result = `
{
   "total_results": 1,
   "total_pages": 1,
   "prev_url": null,
   "next_url": null,
   "resources": [
      {
         "metadata": {
            "guid": "0aea16d1-60d9-4ba2-9e53-ccdfe9ed998f",
            "url": "/v2/shared_domains/0aea16d1-60d9-4ba2-9e53-ccdfe9ed998f",
            "created_at": "2016-06-07T15:21:41Z",
            "updated_at": "2016-08-29T07:36:05Z"
         },
         "entity": {
            "name": "shared-domain.com",
            "router_group_guid": null
         }
      }
   ]
}
`
		})

		It("parses the output of cf curl properly", func() {
			domain, err := cli.GetDefaultSharedDomain(testLogger, cfCli)
			Expect(err).NotTo(HaveOccurred())
			Expect(domain).To(Equal("shared-domain.com"))
		})
	})

	Context("when cf curl returns a no domains", func() {
		BeforeEach(func() {
			result = `
{
   "total_results": 1,
   "total_pages": 1,
   "prev_url": null,
   "next_url": null,
   "resources": [
   ]
}
`
		})

		It("returns an error to the caller", func() {
			_, err := cli.GetDefaultSharedDomain(testLogger, cfCli)
			Expect(err).To(MatchError(cli.ErrNoDomains))
		})
	})

	Context("when cf curl returns an error", func() {
		BeforeEach(func() {
			returnedErr = errors.New("dummy error")
		})

		It("returns the error to the caller", func() {
			_, err := cli.GetDefaultSharedDomain(testLogger, cfCli)
			Expect(err).To(MatchError(returnedErr))
		})
	})

	Context("if the response isn't parseable", func() {
		BeforeEach(func() {
			result = `{`
		})

		It("returns the error to the caller", func() {
			_, err := cli.GetDefaultSharedDomain(testLogger, cfCli)
			Expect(err).To(MatchError(ContainSubstring("JSON")))
		})
	})
})
