package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	"io/ioutil"
	"os"
)

var _ = Describe("Syncer", func() {
	var runner *SyncerRunner

	BeforeEach(func() {
		initConfig()
		runner = NewSyncerRunner()
	})

	JustBeforeEach(func() {
		runner.Start()
	})

	AfterEach(func() {
		runner.KillWithFire()
	})

	It("should start syncer", func() {
		Eventually(runner.Session).Should(Say("syncer.started"))
		Consistently(runner.Session).ShouldNot(Exit())
	})

	Context("with a missing config file", func() {
		BeforeEach(func() {
			runner.configPath = "not-exist"
		})

		It("fails with an error", func() {
			Eventually(runner.Session).Should(Exit(1))
			Expect(runner.Session.Buffer()).To(Say("failed to open config file"))
		})
	})

	Context("with an invalid config file", func() {
		BeforeEach(func() {
			badfile, err := ioutil.TempFile("", "no-content")
			Expect(err).NotTo(HaveOccurred())
			runner.configPath = badfile.Name()
			ioutil.WriteFile(runner.configPath, []byte("not-exist"), os.ModePerm)
		})

		AfterEach(func() {
			os.Remove(runner.configPath)
		})

		It("fails with an error", func() {
			Eventually(runner.Session).Should(Exit(1))
			Expect(runner.Session.Buffer()).To(Say("failed to read config file"))
		})
	})

	Context("with missing/invalid configuration", func() {
		BeforeEach(func() {

			cfg.SynchronizeInterval = -1

			cfg := writeConfig(&cfg)
			runner.configPath = cfg.Name()
		})

		AfterEach(func() {
			os.Remove(runner.configPath)
		})

		It("should fail validation", func() {
			Eventually(runner.Session).Should(Exit(1))
			Expect(runner.Session.Buffer()).To(Say("failed to validate configuration"))
		})
	})

	Context("when connection to policy db fails", func() {
		BeforeEach(func() {
			cfg.Db.PolicyDbUrl = "postgres://wrongUser:wrongPwd@localhost/autoscaler?sslmode=disable"
			cfg := writeConfig(&cfg)
			runner.configPath = cfg.Name()
		})

		AfterEach(func() {
			os.Remove(runner.configPath)
		})

		It("should error", func() {
			Eventually(runner.Session).Should(Exit(1))
			Expect(runner.Session.Buffer()).To(Say("failed to connect policy database"))
		})

	})

	Context("when connection to scheduler db fails", func() {
		BeforeEach(func() {
			cfg.Db.SchedulerDbUrl = "postgres://wrongUser:wrongPwd@localhost/autoscaler?sslmode=disable"
			cfg := writeConfig(&cfg)
			runner.configPath = cfg.Name()
		})

		AfterEach(func() {
			os.Remove(runner.configPath)
		})

		It("should error", func() {
			Eventually(runner.Session).Should(Exit(1))
			Expect(runner.Session.Buffer()).To(Say("failed to connect scheduler database"))
		})

	})

	Context("when an interrupt is sent", func() {
		It("should stop", func() {
			runner.Interrupt()
			Eventually(runner.Session, 5).Should(Exit(130))
		})
	})
})
