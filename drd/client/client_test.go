package client_test

import (
	"errors"

	"code.cloudfoundry.org/bbs/fake_bbs"
	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/diego-stress-tests/drd/client"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var (
		logger        *lagertest.TestLogger
		fakeBBSClient *fake_bbs.FakeClient
	)

	BeforeEach(func() {
		fakeBBSClient = &fake_bbs.FakeClient{}
		logger = lagertest.NewTestLogger("test-client")
	})

	Context("DesiredLRP", func() {
		var (
			expectedDesiredLRPs []*models.DesiredLRP
		)

		BeforeEach(func() {
			expectedDesiredLRPs = []*models.DesiredLRP{
				&models.DesiredLRP{
					ProcessGuid: "test-guid",
					Instances:   1,
				},
			}

			fakeBBSClient.DesiredLRPsReturns(expectedDesiredLRPs, nil)
		})

		It("writes the json representation of the desired LRP to stdout", func() {
			desiredLRPs, err := client.DesiredLRPs(logger, fakeBBSClient, "test-guid")
			Expect(err).NotTo(HaveOccurred())

			Expect(desiredLRPs).To(Equal(expectedDesiredLRPs))
		})
	})

	Context("ActualLRPGroupsForGuid", func() {
		var (
			fakeBBSClient           *fake_bbs.FakeClient
			expectedActualLRPGroups []*models.ActualLRPGroup
		)

		BeforeEach(func() {
			fakeBBSClient = &fake_bbs.FakeClient{}

			expectedActualLRPGroups = []*models.ActualLRPGroup{
				{
					Instance: &models.ActualLRP{
						CrashCount:  15,
						CrashReason: "I need some JSON",
					},
				},
				{
					Evacuating: &models.ActualLRP{
						CrashCount:  7,
						CrashReason: "I need some more JSON",
					},
				},
			}

			fakeBBSClient.ActualLRPGroupsByProcessGuidReturns(expectedActualLRPGroups, nil)
		})

		It("returns the actual lrps for the given guid", func() {
			actualLRPGroups, err := client.ActualLRPGroupsForGuid(logger, fakeBBSClient, "guid")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeBBSClient.ActualLRPGroupsByProcessGuidCallCount()).To(Equal(1))
			_, guid := fakeBBSClient.ActualLRPGroupsByProcessGuidArgsForCall(0)
			Expect(guid).To(Equal("guid"))
			Expect(actualLRPGroups).To(Equal(expectedActualLRPGroups))
		})

		Context("when fetching actual lrp groups fails", func() {
			BeforeEach(func() {
				fakeBBSClient.ActualLRPGroupsByProcessGuidReturns(nil, errors.New("i-failed"))
			})

			It("returns the error", func() {
				_, err := client.ActualLRPGroupsForGuid(logger, fakeBBSClient, "guid")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
