package peformance_setup_test

import (
	"acceptance/helpers"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Prepare test apps based on performance inputs", func() {
	var (
		appName          string
		runningAppsCount int32
		pendingApps      sync.Map
		errors           sync.Map
		itSpecText       string
	)

	AfterEach(func() {
		pendingApps.Range(func(k, v interface{}) bool {
			fmt.Printf("pending app: %s \n", k)
			return true
		})

		errors.Range(func(appName, err interface{}) bool {
			fmt.Printf("errors by app: %s: %s \n", appName, err.(error).Error())
			return true
		})
	})
	BeforeEach(func() {

		wg := sync.WaitGroup{}
		queue := make(chan string)
		workerCount := cfg.Performance.SetupWorkers
		var desiredApps []string

		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go appHandler(queue, &runningAppsCount, &pendingApps, &errors, &wg)
		}
		for i := 0; i < cfg.Performance.AppCount; i++ {
			appName = fmt.Sprintf("node-custom-metric-benchmark-%d", i)
			desiredApps = append(desiredApps, appName)
			pendingApps.Store(appName, 1)
		}
		fmt.Printf("desired app count: %d\n", len(desiredApps))
		appNameGenerator(queue, desiredApps)
		itSpecText = fmt.Sprintf(" should be equal to %d", cfg.Performance.AppCount)
		close(queue)
		fmt.Println("\nWaiting for apps to finish...")
		wg.Wait()
		fmt.Printf("\nTotal Running apps: %d/%d\n", atomic.LoadInt32(&runningAppsCount), cfg.Performance.AppCount)
	})

	Context("App count", func() {
		It(itSpecText, func() {
			Eventually(func() int32 {
				return atomic.LoadInt32(&runningAppsCount)
			},
				300*time.Minute, 5*time.Second).
				Should(BeEquivalentTo(cfg.Performance.AppCount))
		})
	})
})

func appNameGenerator(ch chan<- string, desiredApps []string) {
	for _, app := range desiredApps {
		ch <- app
	}
}

func appHandler(ch <-chan string, runningAppsCount *int32, pendingApps *sync.Map, errors *sync.Map, wg *sync.WaitGroup) {
	defer wg.Done()
	defer GinkgoRecover()

	for appName := range ch {
		fmt.Printf("- pushing app [ %s ] \n", appName)
		pushAppAndBindService(appName, runningAppsCount, pendingApps, errors)
		time.Sleep(time.Millisecond)
	}
}

func pushAppAndBindService(appName string, runningApps *int32, pendingApps *sync.Map, errors *sync.Map) {
	err := helpers.CreateTestAppFromDropletByName(cfg, nodeAppDropletPath, appName, 1)
	if err != nil {
		errors.Store(appName, err)
		return
	}
	policy := helpers.GenerateDynamicScaleOutAndInPolicy(
		1, 2, "test_metric", 500, 500)
	_, err = helpers.GetAppGuid(cfg, appName)
	if err != nil {
		errors.Store(appName, err)
		return
	}
	_, err = helpers.CreatePolicyWithErr(cfg, appName, policy)
	if err != nil {
		errors.Store(appName, err)
		return
	}
	err = helpers.StartAppWithErr(appName, cfg.CfPushTimeoutDuration())
	if err != nil {
		errors.Store(appName, err)
		return
	}
	atomic.AddInt32(runningApps, 1)
	pendingApps.Delete(appName)
	fmt.Printf("  - Running apps: %d/%d - %s\n", atomic.LoadInt32(runningApps), cfg.Performance.AppCount, appName)
}
