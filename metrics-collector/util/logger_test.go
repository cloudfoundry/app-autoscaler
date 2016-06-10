package util_test

import (
	"github.com/cloudfoundry-incubator/app-autoscaler/metrics-collector/config"
	. "github.com/cloudfoundry-incubator/app-autoscaler/metrics-collector/util"
	"github.com/pivotal-golang/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
)

var _ = Describe("Logger", func() {

	Describe("Get loggging level", func() {
		It("should return correct log level", func() {
			Expect(GetLogLevel("DEBUG")).To(Equal(lager.DEBUG))
			Expect(GetLogLevel("INFO")).To(Equal(lager.INFO))
			Expect(GetLogLevel("ERROR")).To(Equal(lager.ERROR))
			Expect(GetLogLevel("FATAL")).To(Equal(lager.FATAL))

			Expect(GetLogLevel("DEbuG")).To(Equal(lager.DEBUG))
			Expect(GetLogLevel("info")).To(Equal(lager.INFO))
			Expect(GetLogLevel("ErROr")).To(Equal(lager.ERROR))
			Expect(GetLogLevel("FaTal")).To(Equal(lager.FATAL))

			Expect(GetLogLevel("NOT-EXIST")).To(Equal(DEFAULT_LOG_LEVEL))
		})
	})

	Describe("Initialize logger", func() {

		var (
			conf *config.LoggingConfig
			err  error
		)

		JustBeforeEach(func() {
			err = InitailizeLogger(conf)
		})

		Context("Logging to stdout only", func() {
			BeforeEach(func() {
				conf = &config.LoggingConfig{
					Level:       "DEBUG",
					LogToStdout: true,
					File:        "",
				}

			})

			It("should not error and return a valid logger", func() {
				Expect(Logger).NotTo(BeNil())
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when logging to an existing directory", func() {
			BeforeEach(func() {
				conf = &config.LoggingConfig{
					Level:       "INFO",
					LogToStdout: false,
					File:        "test_dir",
				}
				os.Mkdir(conf.File, 0744)

			})

			AfterEach(func() {
				os.RemoveAll(conf.File)
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when logging to a new file", func() {

			BeforeEach(func() {
				conf = &config.LoggingConfig{
					Level:       "INFO",
					LogToStdout: false,
					File:        "test_new_file.log",
				}
			})

			AfterEach(func() {
				os.Remove(conf.File)
			})

			It("should not error and return a valid logger ", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(Logger).NotTo(BeNil())
			})
		})

		Context("when logging to an existing file", func() {

			BeforeEach(func() {
				conf = &config.LoggingConfig{
					Level:       "INFO",
					LogToStdout: false,
					File:        "test_existing_file.log",
				}
				os.Create(conf.File)
			})

			AfterEach(func() {
				os.Remove(conf.File)
			})

			It("should not error and return a valid logger ", func() {
				err := InitailizeLogger(conf)
				Expect(err).NotTo(HaveOccurred())
				Expect(Logger).NotTo(BeNil())
			})
		})

	})
})
