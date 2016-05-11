package app

import (
	"acceptance/api"
	"acceptance/config"
	"acceptance/helpers"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("AutoScaler Application", func() {
	var buildpack string
	var appBits string
	var memory string
	var instanceCount int
	var appName string
	var appGUID string
	var instanceName string
	var policy api.Policy

	JustBeforeEach(func() {
		appName = generator.PrefixedRandomName("autoscaler-APP")
		count := strconv.Itoa(instanceCount)
		createApp := cf.Cf("push", appName, "--no-start", "-i", count, "-b", buildpack, "-m", memory, "-p", appBits, "-d", cfg.AppsDomain).Wait(config.DEFAULT_TIMEOUT)
		Expect(createApp).To(Exit(0), "failed creating app")
		// app_helpers.SetBackend(appName)
		guid := cf.Cf("app", appName, "--guid").Wait(config.DEFAULT_TIMEOUT)
		Expect(guid).To(Exit(0))
		appGUID = strings.TrimSpace(string(guid.Out.Contents()))

		instanceName = generator.PrefixedRandomName("scaling-")
		createService := cf.Cf("create-service", cfg.ServiceName, "free", instanceName).Wait(config.DEFAULT_TIMEOUT)
		Expect(createService).To(Exit(0), "failed creating service")

		bindService := cf.Cf("bind-service", appName, instanceName).Wait(config.DEFAULT_TIMEOUT)
		Expect(bindService).To(Exit(0), "failed binding app to service")
	})

	AfterEach(func() {
		if instanceName != "" {
			unbindService := cf.Cf("unbind-service", appName, instanceName).Wait(config.DEFAULT_TIMEOUT)
			Expect(unbindService).To(Exit(0), "failed unbinding app to service")

			deleteService := cf.Cf("delete-service", instanceName, "-f").Wait(config.DEFAULT_TIMEOUT)
			Expect(deleteService).To(Exit(0))
		}

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(config.CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	Context("with a recurring schedule", func() {
		BeforeEach(func() {
			buildpack = cfg.NodejsBuildpackName
			appBits = config.NODE_APP
			memory = "128M"
			instanceCount = 1
		})

		JustBeforeEach(func() {
			schedule, err := ioutil.ReadFile("../assets/file/policy/recurringSchedule.json.template")
			Expect(err).ToNot(HaveOccurred())

			t := time.Now().UTC()
			start := t.Format("15:04") // Hour:Minute
			t = t.Add(5 * time.Minute)
			end := t.Format("15:04") // Hour:Minute

			schedule = bytes.Replace(schedule, []byte("{startTimeValue}"), []byte(start), 1)
			schedule = bytes.Replace(schedule, []byte("{endTimeValue}"), []byte(end), 1)

			policy = api.NewPolicy(cfg.APIUrl, appGUID)
			Expect(cf.Cf("start", appName).Wait(config.CF_PUSH_TIMEOUT)).To(Exit(0))
			waitForNInstancesRunning(appGUID, instanceCount, config.DEFAULT_TIMEOUT)

			statusCode, err := policy.UpdateWithText(string(schedule))
			Expect(err).ToNot(HaveOccurred())
			Expect(statusCode).To(Equal(http.StatusCreated))
		})

		It("will scale out", func() {
			waitForNInstancesRunning(appGUID, instanceCount+1, 90*time.Second)

			code, out, err := api.GetHistory(cfg.APIUrl, appGUID)
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(http.StatusOK))

			var r api.HistoryResponse
			err = json.Unmarshal(out, &r)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.Data).To(HaveLen(1))
			h := r.Data[0]
			Expect(h.InstancesBefore).To(Equal(1))
			Expect(h.InstancesAfter).To(Equal(2))
		})
	})

	Context("with scale by metrics", func() {
		BeforeEach(func() {
			buildpack = cfg.NodejsBuildpackName
			appBits = config.NODE_APP
			memory = "128M"
			instanceCount = 1
		})

		JustBeforeEach(func() {
			policy = api.NewPolicy(cfg.APIUrl, appGUID)
			Expect(cf.Cf("start", appName).Wait(config.CF_PUSH_TIMEOUT)).To(Exit(0))
			waitForNInstancesRunning(appGUID, instanceCount, config.DEFAULT_TIMEOUT)

			schedule, err := ioutil.ReadFile("../assets/file/policy/dynamic.json.template")
			Expect(err).ToNot(HaveOccurred())

			schedule = bytes.Replace(schedule, []byte("{maxCount}"), []byte{'5'}, 1)
			schedule = bytes.Replace(schedule, []byte("{reportInterval}"), []byte(strconv.Itoa(cfg.ReportInterval)), -1)

			statusCode, err := policy.UpdateWithText(string(schedule))
			Expect(err).ToNot(HaveOccurred())
			Expect(statusCode).To(Equal(http.StatusCreated))
		})

		It("scales out", func() {
			totalTime := time.Duration(cfg.ReportInterval*2)*time.Second + 2*time.Minute
			addURL := fmt.Sprintf("https://%s.%s?cmd=add&mode=normal&num=50000", appName, cfg.AppsDomain)
			finishTime := time.Now().Add(totalTime)

			var previousMemory, newMemory, quota uint64
			added := false
			Eventually(func() int {
				// add memory if we are < 50% used
				if previousMemory == 0 || quota/previousMemory > 1 {
					status, _, err := helpers.Curl("-k", "-s", addURL)
					Expect(err).NotTo(HaveOccurred())
					Expect(status).To(Equal(http.StatusOK))
					added = true
				}

				remaining := finishTime.Sub(time.Now())

				if added {
					// wait until memory bumps
					Eventually(func() uint64 {
						newMemory, quota = memoryUsed(appGUID, 0, remaining)
						return newMemory
					}, remaining, 15*time.Second).Should(BeNumerically(">", previousMemory))
					previousMemory = newMemory
				}

				remaining = finishTime.Sub(time.Now())
				return runningInstances(appGUID, remaining)
			}, totalTime, 15*time.Second).Should(BeNumerically(">", instanceCount))

			status, data, err := api.GetHistory(cfg.APIUrl, appGUID)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(http.StatusOK))

			var history api.HistoryResponse
			err = json.Unmarshal(data, &history)
			Expect(err).NotTo(HaveOccurred())
			Expect(history.Data).To(HaveLen(1))
			historyData := history.Data[0]
			Expect(historyData.InstancesBefore).To(Equal(instanceCount))
			Expect(historyData.InstancesAfter).To(Equal(instanceCount + 1))
		})

		Context("with 2 instances", func() {
			BeforeEach(func() {
				memory = "512M"
				instanceCount = 2
			})

			It("scales in", func() {
				totalTime := time.Duration(cfg.ReportInterval*2)*time.Second + time.Minute
				finishTime := time.Now().Add(totalTime)

				// make sure our threshold is < 30%
				Eventually(func() uint64 {
					mem, quota := allMemoryUsed(appGUID, totalTime)
					var total uint64
					for _, m := range mem {
						if m == 0 {
							return math.MaxInt32
						}
						total += m
					}

					return total * 100 / quota
				}, totalTime, 15*time.Second).Should(BeNumerically("<", 30))

				waitForNInstancesRunning(appGUID, instanceCount-1, finishTime.Sub(time.Now()))

				status, data, err := api.GetHistory(cfg.APIUrl, appGUID)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(http.StatusOK))

				var history api.HistoryResponse
				err = json.Unmarshal(data, &history)
				Expect(err).NotTo(HaveOccurred())
				Expect(history.Data).To(HaveLen(1))
				historyData := history.Data[0]
				Expect(historyData.InstancesBefore).To(Equal(instanceCount))
				Expect(historyData.InstancesAfter).To(Equal(instanceCount - 1))
			})
		})
	})
})
