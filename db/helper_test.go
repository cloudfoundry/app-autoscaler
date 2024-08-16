package db_test

import (
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helper", func() {
	var (
		dbUrl    string
		err      error
		database *Database
		certPath string
	)

	Describe("GetConnection", func() {

		JustBeforeEach(func() {
			database, err = GetConnection(dbUrl)
		})
		Context("when mysql query parameters are provided", func() {
			BeforeEach(func() {
				dbUrl = "root@tcp(localhost:3306)/autoscaler?tls=preferred"
			})
			It("returns mysql database object", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(database).To(Equal(&Database{
					DriverName:     "mysql",
					DataSourceName: "root@tcp(localhost:3306)/autoscaler?parseTime=true&tls=preferred",
					OTELAttribute:  semconv.DBSystemMySQL,
				}))
			})
		})

		Context("when mysql query parameters are not provided", func() {
			BeforeEach(func() {
				dbUrl = "root@tcp(localhost:3306)/autoscaler"
			})
			It("returns mysql database object", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(database).To(Equal(&Database{
					DriverName:     "mysql",
					DataSourceName: "root@tcp(localhost:3306)/autoscaler?parseTime=true",
					OTELAttribute:  semconv.DBSystemMySQL,
				}))
			})

		})

		Context("when need to verify mysql server, cert is provided ", func() {
			BeforeEach(func() {
				certPath = "../../../test-certs/api.crt"
				dbUrl = "root@tcp(localhost:3306)/autoscaler?tls=verify-ca&sslrootcert=" + certPath
			})
			It("returns mysql database connection", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(database).To(Equal(&Database{
					DriverName:     "mysql",
					DataSourceName: "root@tcp(localhost:3306)/autoscaler?parseTime=true&tls=verify-ca",
					OTELAttribute:  semconv.DBSystemMySQL,
				}))
			})
		})

		Context("when need to verify mysql server, cert is not provided ", func() {
			BeforeEach(func() {
				dbUrl = "root@tcp(localhost:3306)/autoscaler?tls=verify-ca"
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("sql ca file is not provided"))
			})
		})

		Context("when postgres dburl is provided", func() {
			BeforeEach(func() {
				dbUrl = "postgres://postgres:password@localhost:5432/autoscaler?sslmode=disable"
			})
			It("returns postgres database object", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(database).To(Equal(&Database{
					DriverName:     "pgx",
					DataSourceName: "postgres://postgres:password@localhost:5432/autoscaler?sslmode=disable",
					OTELAttribute:  semconv.DBSystemPostgreSQL,
				}))
			})
		})
	})
})
