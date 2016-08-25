package parser_test

import (
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/diego-stress-tests/arborist/parser"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parser", func() {
	var (
		logger *lagertest.TestLogger
		file   *os.File

		domain, testAppFileContents string
	)

	BeforeEach(func() {
		testAppFileContents = `{
			"succeeded": true,
			"apps": [
			  {
					"app_name": "test_app_1",
					"app_guid": "test_app_1_guid",
					"start": {
						"succeeded": true
				}
				},
				{
					"app_name": "test_app_2",
					"app_guid": "test_app_2_guid",
					"start": {
						"succeeded": true
					}
				},
				{
					"app_name": "test_app_3",
					"app_guid": "test_app_3_guid",
					"start": {
						"succeeded": false
					}
				}
			]
		}`

		logger = lagertest.NewTestLogger("arborist-test")
		domain = "fake-domain.com"
	})

	JustBeforeEach(func() {
		var err error
		file, err = ioutil.TempFile("", "test-app-file")
		Expect(err).NotTo(HaveOccurred())

		n, err := file.WriteString(testAppFileContents)
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(len(testAppFileContents)))
	})

	AfterEach(func() {
		err := os.RemoveAll(file.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	It("reads an app file and returns a list of started app definitions", func() {
		applications, err := parser.ParseAppFile(logger, file.Name(), domain)
		Expect(err).NotTo(HaveOccurred())
		Expect(applications).To(HaveLen(2))
		Expect(applications[0].Name).To(Equal("test_app_1"))
		Expect(applications[1].Name).To(Equal("test_app_2"))
		Expect(applications[0].Url).To(Equal("http://test-app-1.fake-domain.com"))
		Expect(applications[1].Url).To(Equal("http://test-app-2.fake-domain.com"))
	})

	Context("when the json is not valid", func() {
		BeforeEach(func() {
			testAppFileContents = "{{"
		})

		It("returns an error", func() {
			_, err := parser.ParseAppFile(logger, file.Name(), domain)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when the file is not present", func() {
		JustBeforeEach(func() {
			err := os.RemoveAll(file.Name())
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an error", func() {
			_, err := parser.ParseAppFile(logger, file.Name(), domain)
			Expect(err).To(HaveOccurred())
		})
	})
})
