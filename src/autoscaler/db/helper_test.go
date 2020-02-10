package db_test

import (
	. "autoscaler/db"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helper", func() {
	var (
		dbUrl      string
		err        error
		database   *Database
	)
	
	Describe("Connection", func() {
		
		JustBeforeEach(func() {
			database, err = Connection(dbUrl)
		})
		Context("when mysql tls is preferred", func() {
			BeforeEach(func() {
				dbUrl="root@tcp(localhost:3306)/autoscaler?tls=preferred"
			})
			It("returns mysql database object", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(database).To(Equal(&Database{
					DriverName: "mysql",
					DSN: "root@tcp(localhost:3306)/autoscaler?parseTime=true&tls=preferred",
				}))
			})
		})

		Context("when mysql dbburl is invalid", func() {
			BeforeEach(func() {
				dbUrl="root@tcp(localhost:3306)/autoscaler?tls=custom"
			})
			It("should error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("sql ca file is not provided when tls is a custom key"))
			})
		})

		Context("when postgres dburl is provided", func() {
			BeforeEach(func() {
				dbUrl="postgres://postgres:password@localhost:5432/autoscaler?sslmode=disable"
			})
			It("returns postgres database object", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(database).To(Equal(&Database{
					DriverName: "postgres",
					DSN: "postgres://postgres:password@localhost:5432/autoscaler?sslmode=disable",
				}))
			})
		})
	})
})