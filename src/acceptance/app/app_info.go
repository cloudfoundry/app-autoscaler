package app

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type appSummary struct {
	RunningInstances int `json:"running_instances"`
}

func runningInstances(appGUID string, timeout time.Duration) int {
	cmd := cf.Cf("curl", "/v2/apps/"+appGUID+"/summary")
	Expect(cmd.Wait(timeout)).To(Exit(0))

	var summary appSummary
	err := json.Unmarshal(cmd.Out.Contents(), &summary)
	Expect(err).ToNot(HaveOccurred())
	return summary.RunningInstances
}

func waitForNInstancesRunning(appGUID string, instances int, timeout time.Duration) {
	Eventually(func() int {
		return runningInstances(appGUID, timeout)
	}, timeout, 10*time.Second).Should(Equal(instances))
}

type instanceStats struct {
	MemQuota uint64 `json:"mem_quota"`
	Usage    instanceUsage
}

type instanceUsage struct {
	Mem uint64
}

type instanceInfo struct {
	State string
	Stats instanceStats
}

type appStats map[string]*instanceInfo

func memoryUsed(appGUID string, index int, timeout time.Duration) (uint64, uint64) {
	cmd := cf.Cf("curl", "/v2/apps/"+appGUID+"/stats")
	Expect(cmd.Wait(timeout)).To(Exit(0))

	var stats appStats
	err := json.Unmarshal(cmd.Out.Contents(), &stats)
	Expect(err).ToNot(HaveOccurred())

	instance := stats[strconv.Itoa(index)]
	if instance == nil {
		return 0, 0
	}
	return instance.Stats.Usage.Mem, instance.Stats.MemQuota
}

func allMemoryUsed(appGUID string, timeout time.Duration) ([]uint64, uint64) {
	cmd := cf.Cf("curl", "/v2/apps/"+appGUID+"/stats")
	Expect(cmd.Wait(timeout)).To(Exit(0))

	var stats appStats
	err := json.Unmarshal(cmd.Out.Contents(), &stats)
	Expect(err).ToNot(HaveOccurred())

	if len(stats) == 0 {
		return []uint64{}, 0
	}

	mem := make([]uint64, len(stats))
	var quota uint64

	for k, instance := range stats {
		i, err := strconv.Atoi(k)
		Expect(err).NotTo(HaveOccurred())
		mem[i] = instance.Stats.Usage.Mem
		quota = instance.Stats.MemQuota
	}

	return mem, quota
}
