package sqldb_test

import (
	"code.cloudfoundry.org/lager"
	. "db/sqldb"
	"eventgenerator/policy"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"
)

var _ = Describe("PolicySQLDB", func() {
	var (
		pdb      *PolicySQLDB
		url      string
		logger   lager.Logger
		err      error
		appIds   map[string]bool
		policies []*policy.PolicyJson
	)

	BeforeEach(func() {
		logger = lager.NewLogger("policy-sqldb-test")
		url = os.Getenv("DBURL")
	})

	Describe("NewPolicySQLDB", func() {
		JustBeforeEach(func() {
			pdb, err = NewPolicySQLDB(url, logger)
		})

		AfterEach(func() {
			if pdb != nil {
				err = pdb.Close()
				Expect(err).NotTo(HaveOccurred())
			}
		})

		Context("when db url is not correct", func() {
			BeforeEach(func() {
				url = "postgres://non-exist-user:non-exist-password@localhost/autoscaler?sslmode=disable"
			})
			It("should error", func() {
				Expect(err).To(BeAssignableToTypeOf(&pq.Error{}))
			})

		})

		Context("when db url is correct", func() {
			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(pdb).NotTo(BeNil())
			})
		})
	})

	Describe("GetAppIds", func() {
		BeforeEach(func() {
			pdb, err = NewPolicySQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanPolicyTable()
		})

		AfterEach(func() {
			err = pdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			appIds, err = pdb.GetAppIds()
		})

		Context("when policy table is empty", func() {
			It("returns no app ids", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appIds).To(BeEmpty())
			})
		})

		Context("when policy table is not empty", func() {
			BeforeEach(func() {
				insertPolicy("first-app-id")
				insertPolicy("second-app-id")
				insertPolicy("third-app-id")
			})

			It("returns all app ids", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(appIds).To(HaveKey("first-app-id"))
				Expect(appIds).To(HaveKey("second-app-id"))
				Expect(appIds).To(HaveKey("third-app-id"))
			})
		})
	})
	Describe("RetrievePolicies", func() {
		BeforeEach(func() {
			pdb, err = NewPolicySQLDB(url, logger)
			Expect(err).NotTo(HaveOccurred())

			cleanPolicyTable()
		})

		AfterEach(func() {
			err = pdb.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		JustBeforeEach(func() {
			insertPolicy("first-app-id")
			insertPolicy("second-app-id")
			insertPolicy("third-app-id")
			policies, err = pdb.RetrievePolicies()
		})

		Context("when retriving all the policies)", func() {
			It("returns all the policies", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(policies).To(ConsistOf(
					&policy.PolicyJson{
						AppId: "first-app-id",
						PolicyStr: `
		{
 			"instance_min_count": 1,
  			"instance_max_count": 5
		}`,
					},
					&policy.PolicyJson{
						AppId: "second-app-id",
						PolicyStr: `
		{
 			"instance_min_count": 1,
  			"instance_max_count": 5
		}`,
					},
					&policy.PolicyJson{
						AppId: "third-app-id",
						PolicyStr: `
		{
 			"instance_min_count": 1,
  			"instance_max_count": 5
		}`,
					},
				))
			})
		})
	})
})
