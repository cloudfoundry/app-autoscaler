package run_performance_test

import (
	"acceptance/helpers"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gmeasure"
)

const pollTime = 15 * time.Second
const timeOutDuration = 20 * time.Minute
const desiredScalingTime = 3 * time.Hour
const maxAppInstancesCount = 2

var _ = Describe("Scale-out and scale-in (eg: 30%) percentage of apps", Ordered, func() {
	var (
		percentageToScale      int
		appCount               int
		actualAppsToScaleCount int
		scaledInAppsCount      atomic.Int32
		scaledOutAppsCount     atomic.Int32
		pendingScaleOuts       sync.Map
		pendingScaleIns        sync.Map
		scaleOutApps           sync.Map
		scaleInApps            sync.Map
		startedApps            []helpers.AppInfo
		allApps                []helpers.AppInfo
		samplingConfig         gmeasure.SamplingConfig
		experiment             = gmeasure.NewExperiment("Scaling Benchmark")
	)
	AfterAll(func() {
		fmt.Println("\n\nSummary...")
		fmt.Print("\nScale-Out Errors\n")
		pendingScaleOuts.Range(func(appName, appGuid interface{}) bool {
			fmt.Printf("scale-out app error: %s: %s \n", appName, appGuid)
			return true
		})
		fmt.Print("\nScale-In Errors\n")
		pendingScaleIns.Range(func(appName, appGuid interface{}) bool {
			fmt.Printf("scale-in app error: %s: %s \n", appName, appGuid)
			return true
		})
	})

	BeforeEach(func() {
		orgGuid := helpers.GetOrgGuid(cfg, cfg.ExistingOrganization)
		spaceGuid := helpers.GetSpaceGuid(cfg, orgGuid)
		startedApps = helpers.GetAllStartedApp(cfg, orgGuid, spaceGuid, "node-custom-metric-benchmark")

		percentageToScale, appCount = cfg.Performance.PercentageToScale, cfg.Performance.AppCount
		if percentageToScale < 0 || percentageToScale > 100 {
			err := fmt.Errorf(
				"given scaling percentage not in [0, 100] which does not make sense: percentageToScale = %d",
				percentageToScale)
			Fail(err.Error())
		}
	})

	Context("when scaling out by custom metrics", func() {
		JustBeforeEach(func() {
			desiredAppsToScaleCount := calculateAppsToScaleCount(appCount, percentageToScale)
			// Now calculate desiredAppsToScaleCount based on the actual startedApps
			actualAppsToScaleCount = calculateAppsToScaleCount(len(startedApps), percentageToScale)
			fmt.Printf("\n\nDesired Scaling %d apps \n", desiredAppsToScaleCount)
			fmt.Printf("Actual Scaling %d apps (based on successful apps pushed) \n\n", actualAppsToScaleCount)

			samplingConfig = gmeasure.SamplingConfig{
				N:           actualAppsToScaleCount,
				NumParallel: 20, // number of parallel nodes/processes to execute at a time
				// e.g. 20 scaleout will run on 20 nodes
				Duration: 10 * time.Hour,
			}
		})
		It("should scale out", Label("measurement"), func() {
			AddReportEntry(experiment.Name, experiment)
			wg := sync.WaitGroup{}
			wg.Add(samplingConfig.N)

			experiment.Sample(func(workerIndex int) {
				defer GinkgoRecover()
				defer wg.Done()

				tasksWG := sync.WaitGroup{}
				tasksWG.Add(1)
				appName := startedApps[workerIndex].Name
				appGUID := startedApps[workerIndex].Guid

				pendingScaleOuts.LoadOrStore(appGUID, startedApps[workerIndex])
				experiment.MeasureDuration("scale-out",
					scaleOutApp(appName, appGUID, &scaleOutApps, &pendingScaleOuts, &scaledOutAppsCount,
						actualAppsToScaleCount, workerIndex, &tasksWG))

				tasksWG.Wait()

			}, samplingConfig)

			fmt.Printf("Waiting for scale-out workers to finish scaling. This may take few minutes...\n\n")
			wg.Wait()
			fmt.Printf("\nTotal scale out apps: %d/%d\n", scaledOutAppsCount.Load(), actualAppsToScaleCount)
			// append all apps in the apps from scale-out Map
			allApps = buildAppsList(&scaleOutApps)
		})

	})

	Context("trigger failed scale-outs", func() {
		It("measures scale-outs", func() {
			//Hack: to check for any remaining apps which were scaled out (in cf) but not reported somehow.
			// if len(pendingScaleOuts) > 0, trigger scale out again for such apps and create a new report entry
			pendingScaleOutsLen := int(lenOfSyncMap(&pendingScaleOuts))
			if pendingScaleOutsLen == 0 {
				return
			}
			fmt.Printf("\nlen of pendingScaleOutsLen %d\n", pendingScaleOutsLen)
			// continue to scale-out failed apps
			apps := buildAppsList(&pendingScaleOuts)

			wg := sync.WaitGroup{}
			wg.Add(len(apps))
			AddReportEntry(experiment.Name, experiment)

			experiment.Sample(func(workerIndex int) {
				defer GinkgoRecover()
				defer wg.Done()

				tasksWG := sync.WaitGroup{}
				tasksWG.Add(1)

				appInfo := apps[workerIndex]
				appName := appInfo.Name
				appGUID := appInfo.Guid

				instances, err := helpers.RunningInstances(appGUID, 10*time.Minute)
				if err != nil {
					err = fmt.Errorf("	error fetching running instances for app %s %s %w\n", appName, appGUID, err)
					fmt.Printf("%s", err.Error())
					return
				}
				if instances >= maxAppInstancesCount { // apps are already scaled-out
					experiment.MeasureDuration("scale-out", func() {
						scaleOutApps.LoadOrStore(appGUID, helpers.AppInfo{Name: appName, Guid: appGUID})
						scaledOutAppsCount.Add(1)
						pendingScaleOuts.LoadAndDelete(appGUID)
						fmt.Printf("worker %d -  with App %s %s already has two or more instances\n\n", workerIndex, appName, appGUID)
						tasksWG.Done()
					})
				} else { //perform scale-out again for failed apps
					experiment.MeasureDuration("scale-out-2",
						scaleOutApp(appName, appGUID, &scaleOutApps, &pendingScaleOuts, &scaledOutAppsCount,
							actualAppsToScaleCount, workerIndex, &tasksWG))
				}
				tasksWG.Wait()
			}, gmeasure.SamplingConfig{
				N:           len(apps),
				NumParallel: len(apps),
				Duration:    1 * time.Hour,
			})
			fmt.Printf("Waiting for more scale-out workers to finish scaling. This may take few minutes...\n\n")
			wg.Wait()
			//build slice with all scaleout apps so that scale-In can happen on allApps
			allApps = append(allApps, apps...)
			fmt.Printf("\nTotal scale out apps: %d/%d\n", scaledOutAppsCount.Load(), actualAppsToScaleCount)

		})
	})
	Context("scale-out results", func() {
		It("wait for scale-out results", func() {
			Eventually(func() int32 {
				count := scaledOutAppsCount.Load()
				fmt.Printf("current scaledOutAppsCount %d\n", count)
				return count
			}).WithPolling(10 * time.Second).
				WithTimeout(desiredScalingTime).
				Should(BeEquivalentTo(actualAppsToScaleCount))
		})
	})

	Context("when scaling In by custom metrics", func() {
		JustBeforeEach(func() {
			fmt.Printf("\nScale-In start for %d apps....\n", scaledOutAppsCount.Load())
			samplingConfig.N = int(scaledOutAppsCount.Load())
		})
		It("should scale in", Label("measurement"), func() {
			AddReportEntry(experiment.Name, experiment)
			wg := sync.WaitGroup{}
			wg.Add(samplingConfig.N)

			experiment.Sample(func(workerIndex int) {
				defer GinkgoRecover()
				defer wg.Done()

				scaledOutApps := allApps[workerIndex]
				appName := scaledOutApps.Name
				appGUID := scaledOutApps.Guid
				tasksWG := sync.WaitGroup{}
				tasksWG.Add(1)

				pendingScaleIns.LoadOrStore(appGUID, scaledOutApps)
				experiment.MeasureDuration("scale-in",
					scaleInApp(appName, appGUID, &scaleInApps, &pendingScaleIns,
						&scaledInAppsCount, actualAppsToScaleCount, workerIndex, &tasksWG))
				tasksWG.Wait()

			}, samplingConfig)

			fmt.Printf("Waiting for scale-In workers to finish scaling.This may take few minutes...\n\n")
			wg.Wait()
			fmt.Printf("\nTotal scale In apps: %d/%d\n", scaledInAppsCount.Load(), actualAppsToScaleCount)

		})
	})
	Context("trigger failed scale-Ins", func() {
		It("measures scale-Ins", func() {
			//Hack: to check for any remaining apps which were scaled In (in cf) but not reported somehow.
			// if len(pendingScaleIns) > 0, trigger scale out again for such apps and create a new report entry

			pendingScaleInsLen := int(lenOfSyncMap(&pendingScaleIns))
			if pendingScaleInsLen == 0 {
				return
			}
			fmt.Printf("\nlen of pendingScaleIns %d\n", pendingScaleInsLen)
			// continue to scale-in failed apps
			var apps []helpers.AppInfo
			pendingScaleIns.Range(func(appGuid, infoObj interface{}) bool {
				appInfo := infoObj.(helpers.AppInfo)
				fmt.Printf("scale-in app error: %s: %s \n", appGuid, appInfo.Name)
				apps = append(apps, appInfo)
				return true
			})

			wg := sync.WaitGroup{}
			wg.Add(len(apps))
			AddReportEntry(experiment.Name, experiment)

			experiment.Sample(func(workerIndex int) {
				defer GinkgoRecover()
				defer wg.Done()

				tasksWG := sync.WaitGroup{}
				tasksWG.Add(1)

				appInfo := apps[workerIndex]
				appName := appInfo.Name
				appGUID := appInfo.Guid

				instances, err := helpers.RunningInstances(appGUID, 10*time.Minute)
				if err != nil {
					err = fmt.Errorf("	"+
						"scale-in - error fetching running instances for app %s %s %w\n", appName, appGUID, err)
					fmt.Printf("%s", err.Error())
					return
				}
				if instances >= maxAppInstancesCount { // apps are already scaled-in
					experiment.MeasureDuration("scale-in-2",
						scaleInApp(appName, appGUID, &scaleInApps, &pendingScaleIns,
							&scaledInAppsCount, actualAppsToScaleCount, workerIndex, &tasksWG))

				} else { //perform scale-out again for failed apps
					experiment.MeasureDuration("scale-In-2", func() {
						scaleInApps.LoadOrStore(appGUID, helpers.AppInfo{Name: appName, Guid: appGUID})
						scaledInAppsCount.Add(1)
						pendingScaleIns.LoadAndDelete(appGUID)
						fmt.Printf("worker %d -  with App %s %s already has two instances\n\n", workerIndex, appName, appGUID)
						tasksWG.Done()
					})
				}
				tasksWG.Wait()

			}, gmeasure.SamplingConfig{
				N:           len(apps),
				NumParallel: len(apps),
				Duration:    1 * time.Hour,
			})
			fmt.Printf("Waiting for more scale-out workers to finish scaling. This may take few minutes...\n\n")
			wg.Wait()
			fmt.Printf("\nTotal scale out apps: %d/%d\n", scaledInAppsCount.Load(), actualAppsToScaleCount)

		})
	})

	Context("scale-In results", func() {
		It("wait for scale-in results", func() {

			Eventually(func() int32 {
				count := scaledInAppsCount.Load()
				fmt.Printf("current scaledInAppsCount %d\n", count)
				return count
			}).WithPolling(10 * time.Second).
				WithTimeout(desiredScalingTime).
				Should(BeEquivalentTo(actualAppsToScaleCount))

			checkMedianDurationFor(experiment, "scale-out")
			checkMedianDurationFor(experiment, "scale-in")
		})
	})
})

func calculateAppsToScaleCount(appCount int, percentageToScale int) int {
	appsToScaleCount := appCount * percentageToScale / 100
	Expect(appsToScaleCount).To(BeNumerically(">", 0),
		fmt.Sprintf("%d percent of %d must round up to 1 or more app(s)", percentageToScale, appCount))
	return appsToScaleCount
}

func buildAppsList(appsMap *sync.Map) []helpers.AppInfo {
	var allApps []helpers.AppInfo
	appsMap.Range(func(appGuid interface{}, infoObj interface{}) bool {
		appInfo := infoObj.(helpers.AppInfo)
		allApps = append(allApps, appInfo)
		return true
	})
	return allApps
}

func scaleOutApp(appName string, appGUID string, scaleOutApps *sync.Map,
	pendingScaleOuts *sync.Map, scaledOutAppsCount *atomic.Int32,
	actualAppsToScaleCount int, workerIndex int, wg *sync.WaitGroup) func() {
	return func() {
		scaleOut := func() (int, error) {
			// Q. why sending post request to autoscaler after every pollTime.
			// A. It is observed that sometime cf does not pick the cf scale event. Therefore,
			// sending the metric again(and again) is the way to go at the moment
			cmdOutput := helpers.SendMetricMTLS(cfg, appGUID, appName, 550, 5*time.Minute)
			GinkgoWriter.Printf("worker %d - scale-out %s with App %s %s\n",
				workerIndex, cmdOutput, appName, appGUID)
			instances, err := helpers.RunningInstances(appGUID, 10*time.Minute)
			if err != nil {
				err = fmt.Errorf("	error running instances for app %s %s %w\n", appName, appGUID, err)
				fmt.Printf("%s", err.Error())
				return 0, err
			}
			return instances, nil
		}
		Eventually(scaleOut).
			WithPolling(pollTime).
			WithTimeout(timeOutDuration).
			Should(Equal(2),
				fmt.Sprintf("Failed to scale out app: %s", appName))
		scaledOutAppsCount.Add(1)
		scaleOutApps.LoadOrStore(appGUID, helpers.AppInfo{Name: appName, Guid: appGUID})
		fmt.Printf("Scaled-Out apps: %d/%d\n", scaledOutAppsCount.Load(), actualAppsToScaleCount)
		pendingScaleOuts.LoadAndDelete(appGUID)

		defer wg.Done()
	}
}

func scaleInApp(appName string, appGUID string, scaleInApps *sync.Map, pendingScaleIns *sync.Map,
	scaledInAppsCount *atomic.Int32, actualAppsToScaleCount int, workerIndex int, wg *sync.WaitGroup) func() {
	return func() {
		scaleIn := func() (int, error) {
			cmdOutput := helpers.SendMetricMTLS(cfg, appGUID, appName, 100, 5*time.Minute)
			GinkgoWriter.Printf("worker %d - scale-in %s with App %s %s\n",
				workerIndex, cmdOutput, appName, appGUID)
			instances, err := helpers.RunningInstances(appGUID, 10*time.Minute)
			if err != nil {
				err = fmt.Errorf("	error running instances for app %s %s %w\n", appName, appGUID, err)
				fmt.Printf("%s", err.Error())
				return 0, err
			}
			return instances, nil
		}
		Eventually(scaleIn).
			WithPolling(pollTime).
			WithTimeout(timeOutDuration).
			Should(Equal(1),
				fmt.Sprintf("Failed to scale in app: %s", appName))

		scaledInAppsCount.Add(1)
		scaleInApps.LoadOrStore(appGUID, helpers.AppInfo{Name: appName, Guid: appGUID})
		fmt.Printf("Scaled-In apps: %d/%d\n", scaledInAppsCount.Load(), actualAppsToScaleCount)
		pendingScaleIns.LoadAndDelete(appGUID)

		defer wg.Done()
	}
}

func checkMedianDurationFor(experiment *gmeasure.Experiment, statName string) {
	stats := experiment.GetStats(statName)
	medianDuration := stats.DurationFor(gmeasure.StatMedian)
	fmt.Printf("\nMedian duration for %s: %d", statName, medianDuration)
}

func lenOfSyncMap(m *sync.Map) int32 {
	var counter atomic.Int32
	m.Range(func(_ any, _ any) bool {
		counter.Add(1)
		return true
	})
	return counter.Load()
}
