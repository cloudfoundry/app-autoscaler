// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"autoscaler/db"
	"autoscaler/healthendpoint"
	"autoscaler/models"
	"sync"
	"time"

	"code.cloudfoundry.org/clock"
)

type FakePolicyDB struct {
	GetAppIdsStub        func() (map[string]bool, error)
	getAppIdsMutex       sync.RWMutex
	getAppIdsArgsForCall []struct{}
	getAppIdsReturns     struct {
		result1 map[string]bool
		result2 error
	}
	getAppIdsReturnsOnCall map[int]struct {
		result1 map[string]bool
		result2 error
	}
	GetAppPolicyStub        func(appId string) (*models.ScalingPolicy, error)
	getAppPolicyMutex       sync.RWMutex
	getAppPolicyArgsForCall []struct {
		appId string
	}
	getAppPolicyReturns struct {
		result1 *models.ScalingPolicy
		result2 error
	}
	getAppPolicyReturnsOnCall map[int]struct {
		result1 *models.ScalingPolicy
		result2 error
	}
	RetrievePoliciesStub        func() ([]*models.PolicyJson, error)
	retrievePoliciesMutex       sync.RWMutex
	retrievePoliciesArgsForCall []struct{}
	retrievePoliciesReturns     struct {
		result1 []*models.PolicyJson
		result2 error
	}
	retrievePoliciesReturnsOnCall map[int]struct {
		result1 []*models.PolicyJson
		result2 error
	}
	CloseStub        func() error
	closeMutex       sync.RWMutex
	closeArgsForCall []struct{}
	closeReturns     struct {
		result1 error
	}
	closeReturnsOnCall map[int]struct {
		result1 error
	}
	EmitHealthMetricsStub        func(h healthendpoint.Health, cclock clock.Clock, interval time.Duration)
	emitHealthMetricsMutex       sync.RWMutex
	emitHealthMetricsArgsForCall []struct {
		h        healthendpoint.Health
		cclock   clock.Clock
		interval time.Duration
	}
	GetCustomMetricsCredsStub        func(appId string) (string, string, error)
	getCustomMetricsCredsMutex       sync.RWMutex
	getCustomMetricsCredsArgsForCall []struct {
		appId string
	}
	getCustomMetricsCredsReturns struct {
		result1 string
		result2 string
		result3 error
	}
	getCustomMetricsCredsReturnsOnCall map[int]struct {
		result1 string
		result2 string
		result3 error
	}
	ValidateCustomMetricsCredsStub        func(appId string, username string, password string) bool
	validateCustomMetricsCredsMutex       sync.RWMutex
	validateCustomMetricsCredsArgsForCall []struct {
		appId    string
		username string
		password string
	}
	validateCustomMetricsCredsReturns struct {
		result1 bool
	}
	validateCustomMetricsCredsReturnsOnCall map[int]struct {
		result1 bool
	}
	ValidateCustomMetricTypesStub        func(appId string, metricsConsumer *models.MetricsConsumer) (bool, error)
	validateCustomMetricTypesMutex       sync.RWMutex
	validateCustomMetricTypesArgsForCall []struct {
		appId           string
		metricsConsumer *models.MetricsConsumer
	}
	validateCustomMetricTypesReturns struct {
		result1 bool
		result2 error
	}
	validateCustomMetricTypesReturnsOnCall map[int]struct {
		result1 bool
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakePolicyDB) GetAppIds() (map[string]bool, error) {
	fake.getAppIdsMutex.Lock()
	ret, specificReturn := fake.getAppIdsReturnsOnCall[len(fake.getAppIdsArgsForCall)]
	fake.getAppIdsArgsForCall = append(fake.getAppIdsArgsForCall, struct{}{})
	fake.recordInvocation("GetAppIds", []interface{}{})
	fake.getAppIdsMutex.Unlock()
	if fake.GetAppIdsStub != nil {
		return fake.GetAppIdsStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.getAppIdsReturns.result1, fake.getAppIdsReturns.result2
}

func (fake *FakePolicyDB) GetAppIdsCallCount() int {
	fake.getAppIdsMutex.RLock()
	defer fake.getAppIdsMutex.RUnlock()
	return len(fake.getAppIdsArgsForCall)
}

func (fake *FakePolicyDB) GetAppIdsReturns(result1 map[string]bool, result2 error) {
	fake.GetAppIdsStub = nil
	fake.getAppIdsReturns = struct {
		result1 map[string]bool
		result2 error
	}{result1, result2}
}

func (fake *FakePolicyDB) GetAppIdsReturnsOnCall(i int, result1 map[string]bool, result2 error) {
	fake.GetAppIdsStub = nil
	if fake.getAppIdsReturnsOnCall == nil {
		fake.getAppIdsReturnsOnCall = make(map[int]struct {
			result1 map[string]bool
			result2 error
		})
	}
	fake.getAppIdsReturnsOnCall[i] = struct {
		result1 map[string]bool
		result2 error
	}{result1, result2}
}

func (fake *FakePolicyDB) GetAppPolicy(appId string) (*models.ScalingPolicy, error) {
	fake.getAppPolicyMutex.Lock()
	ret, specificReturn := fake.getAppPolicyReturnsOnCall[len(fake.getAppPolicyArgsForCall)]
	fake.getAppPolicyArgsForCall = append(fake.getAppPolicyArgsForCall, struct {
		appId string
	}{appId})
	fake.recordInvocation("GetAppPolicy", []interface{}{appId})
	fake.getAppPolicyMutex.Unlock()
	if fake.GetAppPolicyStub != nil {
		return fake.GetAppPolicyStub(appId)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.getAppPolicyReturns.result1, fake.getAppPolicyReturns.result2
}

func (fake *FakePolicyDB) GetAppPolicyCallCount() int {
	fake.getAppPolicyMutex.RLock()
	defer fake.getAppPolicyMutex.RUnlock()
	return len(fake.getAppPolicyArgsForCall)
}

func (fake *FakePolicyDB) GetAppPolicyArgsForCall(i int) string {
	fake.getAppPolicyMutex.RLock()
	defer fake.getAppPolicyMutex.RUnlock()
	return fake.getAppPolicyArgsForCall[i].appId
}

func (fake *FakePolicyDB) GetAppPolicyReturns(result1 *models.ScalingPolicy, result2 error) {
	fake.GetAppPolicyStub = nil
	fake.getAppPolicyReturns = struct {
		result1 *models.ScalingPolicy
		result2 error
	}{result1, result2}
}

func (fake *FakePolicyDB) GetAppPolicyReturnsOnCall(i int, result1 *models.ScalingPolicy, result2 error) {
	fake.GetAppPolicyStub = nil
	if fake.getAppPolicyReturnsOnCall == nil {
		fake.getAppPolicyReturnsOnCall = make(map[int]struct {
			result1 *models.ScalingPolicy
			result2 error
		})
	}
	fake.getAppPolicyReturnsOnCall[i] = struct {
		result1 *models.ScalingPolicy
		result2 error
	}{result1, result2}
}

func (fake *FakePolicyDB) RetrievePolicies() ([]*models.PolicyJson, error) {
	fake.retrievePoliciesMutex.Lock()
	ret, specificReturn := fake.retrievePoliciesReturnsOnCall[len(fake.retrievePoliciesArgsForCall)]
	fake.retrievePoliciesArgsForCall = append(fake.retrievePoliciesArgsForCall, struct{}{})
	fake.recordInvocation("RetrievePolicies", []interface{}{})
	fake.retrievePoliciesMutex.Unlock()
	if fake.RetrievePoliciesStub != nil {
		return fake.RetrievePoliciesStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.retrievePoliciesReturns.result1, fake.retrievePoliciesReturns.result2
}

func (fake *FakePolicyDB) RetrievePoliciesCallCount() int {
	fake.retrievePoliciesMutex.RLock()
	defer fake.retrievePoliciesMutex.RUnlock()
	return len(fake.retrievePoliciesArgsForCall)
}

func (fake *FakePolicyDB) RetrievePoliciesReturns(result1 []*models.PolicyJson, result2 error) {
	fake.RetrievePoliciesStub = nil
	fake.retrievePoliciesReturns = struct {
		result1 []*models.PolicyJson
		result2 error
	}{result1, result2}
}

func (fake *FakePolicyDB) RetrievePoliciesReturnsOnCall(i int, result1 []*models.PolicyJson, result2 error) {
	fake.RetrievePoliciesStub = nil
	if fake.retrievePoliciesReturnsOnCall == nil {
		fake.retrievePoliciesReturnsOnCall = make(map[int]struct {
			result1 []*models.PolicyJson
			result2 error
		})
	}
	fake.retrievePoliciesReturnsOnCall[i] = struct {
		result1 []*models.PolicyJson
		result2 error
	}{result1, result2}
}

func (fake *FakePolicyDB) Close() error {
	fake.closeMutex.Lock()
	ret, specificReturn := fake.closeReturnsOnCall[len(fake.closeArgsForCall)]
	fake.closeArgsForCall = append(fake.closeArgsForCall, struct{}{})
	fake.recordInvocation("Close", []interface{}{})
	fake.closeMutex.Unlock()
	if fake.CloseStub != nil {
		return fake.CloseStub()
	}
	if specificReturn {
		return ret.result1
	}
	return fake.closeReturns.result1
}

func (fake *FakePolicyDB) CloseCallCount() int {
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	return len(fake.closeArgsForCall)
}

func (fake *FakePolicyDB) CloseReturns(result1 error) {
	fake.CloseStub = nil
	fake.closeReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakePolicyDB) CloseReturnsOnCall(i int, result1 error) {
	fake.CloseStub = nil
	if fake.closeReturnsOnCall == nil {
		fake.closeReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.closeReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakePolicyDB) EmitHealthMetrics(h healthendpoint.Health, cclock clock.Clock, interval time.Duration) {
	fake.emitHealthMetricsMutex.Lock()
	fake.emitHealthMetricsArgsForCall = append(fake.emitHealthMetricsArgsForCall, struct {
		h        healthendpoint.Health
		cclock   clock.Clock
		interval time.Duration
	}{h, cclock, interval})
	fake.recordInvocation("EmitHealthMetrics", []interface{}{h, cclock, interval})
	fake.emitHealthMetricsMutex.Unlock()
	if fake.EmitHealthMetricsStub != nil {
		fake.EmitHealthMetricsStub(h, cclock, interval)
	}
}

func (fake *FakePolicyDB) EmitHealthMetricsCallCount() int {
	fake.emitHealthMetricsMutex.RLock()
	defer fake.emitHealthMetricsMutex.RUnlock()
	return len(fake.emitHealthMetricsArgsForCall)
}

func (fake *FakePolicyDB) EmitHealthMetricsArgsForCall(i int) (healthendpoint.Health, clock.Clock, time.Duration) {
	fake.emitHealthMetricsMutex.RLock()
	defer fake.emitHealthMetricsMutex.RUnlock()
	return fake.emitHealthMetricsArgsForCall[i].h, fake.emitHealthMetricsArgsForCall[i].cclock, fake.emitHealthMetricsArgsForCall[i].interval
}

func (fake *FakePolicyDB) GetCustomMetricsCreds(appId string) (string, string, error) {
	fake.getCustomMetricsCredsMutex.Lock()
	ret, specificReturn := fake.getCustomMetricsCredsReturnsOnCall[len(fake.getCustomMetricsCredsArgsForCall)]
	fake.getCustomMetricsCredsArgsForCall = append(fake.getCustomMetricsCredsArgsForCall, struct {
		appId string
	}{appId})
	fake.recordInvocation("GetCustomMetricsCreds", []interface{}{appId})
	fake.getCustomMetricsCredsMutex.Unlock()
	if fake.GetCustomMetricsCredsStub != nil {
		return fake.GetCustomMetricsCredsStub(appId)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return fake.getCustomMetricsCredsReturns.result1, fake.getCustomMetricsCredsReturns.result2, fake.getCustomMetricsCredsReturns.result3
}

func (fake *FakePolicyDB) GetCustomMetricsCredsCallCount() int {
	fake.getCustomMetricsCredsMutex.RLock()
	defer fake.getCustomMetricsCredsMutex.RUnlock()
	return len(fake.getCustomMetricsCredsArgsForCall)
}

func (fake *FakePolicyDB) GetCustomMetricsCredsArgsForCall(i int) string {
	fake.getCustomMetricsCredsMutex.RLock()
	defer fake.getCustomMetricsCredsMutex.RUnlock()
	return fake.getCustomMetricsCredsArgsForCall[i].appId
}

func (fake *FakePolicyDB) GetCustomMetricsCredsReturns(result1 string, result2 string, result3 error) {
	fake.GetCustomMetricsCredsStub = nil
	fake.getCustomMetricsCredsReturns = struct {
		result1 string
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakePolicyDB) GetCustomMetricsCredsReturnsOnCall(i int, result1 string, result2 string, result3 error) {
	fake.GetCustomMetricsCredsStub = nil
	if fake.getCustomMetricsCredsReturnsOnCall == nil {
		fake.getCustomMetricsCredsReturnsOnCall = make(map[int]struct {
			result1 string
			result2 string
			result3 error
		})
	}
	fake.getCustomMetricsCredsReturnsOnCall[i] = struct {
		result1 string
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakePolicyDB) ValidateCustomMetricsCreds(appId string, username string, password string) bool {
	fake.validateCustomMetricsCredsMutex.Lock()
	ret, specificReturn := fake.validateCustomMetricsCredsReturnsOnCall[len(fake.validateCustomMetricsCredsArgsForCall)]
	fake.validateCustomMetricsCredsArgsForCall = append(fake.validateCustomMetricsCredsArgsForCall, struct {
		appId    string
		username string
		password string
	}{appId, username, password})
	fake.recordInvocation("ValidateCustomMetricsCreds", []interface{}{appId, username, password})
	fake.validateCustomMetricsCredsMutex.Unlock()
	if fake.ValidateCustomMetricsCredsStub != nil {
		return fake.ValidateCustomMetricsCredsStub(appId, username, password)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.validateCustomMetricsCredsReturns.result1
}

func (fake *FakePolicyDB) ValidateCustomMetricsCredsCallCount() int {
	fake.validateCustomMetricsCredsMutex.RLock()
	defer fake.validateCustomMetricsCredsMutex.RUnlock()
	return len(fake.validateCustomMetricsCredsArgsForCall)
}

func (fake *FakePolicyDB) ValidateCustomMetricsCredsArgsForCall(i int) (string, string, string) {
	fake.validateCustomMetricsCredsMutex.RLock()
	defer fake.validateCustomMetricsCredsMutex.RUnlock()
	return fake.validateCustomMetricsCredsArgsForCall[i].appId, fake.validateCustomMetricsCredsArgsForCall[i].username, fake.validateCustomMetricsCredsArgsForCall[i].password
}

func (fake *FakePolicyDB) ValidateCustomMetricsCredsReturns(result1 bool) {
	fake.ValidateCustomMetricsCredsStub = nil
	fake.validateCustomMetricsCredsReturns = struct {
		result1 bool
	}{result1}
}

func (fake *FakePolicyDB) ValidateCustomMetricsCredsReturnsOnCall(i int, result1 bool) {
	fake.ValidateCustomMetricsCredsStub = nil
	if fake.validateCustomMetricsCredsReturnsOnCall == nil {
		fake.validateCustomMetricsCredsReturnsOnCall = make(map[int]struct {
			result1 bool
		})
	}
	fake.validateCustomMetricsCredsReturnsOnCall[i] = struct {
		result1 bool
	}{result1}
}

func (fake *FakePolicyDB) ValidateCustomMetricTypes(appId string, metricsConsumer *models.MetricsConsumer) (bool, error) {
	fake.validateCustomMetricTypesMutex.Lock()
	ret, specificReturn := fake.validateCustomMetricTypesReturnsOnCall[len(fake.validateCustomMetricTypesArgsForCall)]
	fake.validateCustomMetricTypesArgsForCall = append(fake.validateCustomMetricTypesArgsForCall, struct {
		appId           string
		metricsConsumer *models.MetricsConsumer
	}{appId, metricsConsumer})
	fake.recordInvocation("ValidateCustomMetricTypes", []interface{}{appId, metricsConsumer})
	fake.validateCustomMetricTypesMutex.Unlock()
	if fake.ValidateCustomMetricTypesStub != nil {
		return fake.ValidateCustomMetricTypesStub(appId, metricsConsumer)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.validateCustomMetricTypesReturns.result1, fake.validateCustomMetricTypesReturns.result2
}

func (fake *FakePolicyDB) ValidateCustomMetricTypesCallCount() int {
	fake.validateCustomMetricTypesMutex.RLock()
	defer fake.validateCustomMetricTypesMutex.RUnlock()
	return len(fake.validateCustomMetricTypesArgsForCall)
}

func (fake *FakePolicyDB) ValidateCustomMetricTypesArgsForCall(i int) (string, *models.MetricsConsumer) {
	fake.validateCustomMetricTypesMutex.RLock()
	defer fake.validateCustomMetricTypesMutex.RUnlock()
	return fake.validateCustomMetricTypesArgsForCall[i].appId, fake.validateCustomMetricTypesArgsForCall[i].metricsConsumer
}

func (fake *FakePolicyDB) ValidateCustomMetricTypesReturns(result1 bool, result2 error) {
	fake.ValidateCustomMetricTypesStub = nil
	fake.validateCustomMetricTypesReturns = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakePolicyDB) ValidateCustomMetricTypesReturnsOnCall(i int, result1 bool, result2 error) {
	fake.ValidateCustomMetricTypesStub = nil
	if fake.validateCustomMetricTypesReturnsOnCall == nil {
		fake.validateCustomMetricTypesReturnsOnCall = make(map[int]struct {
			result1 bool
			result2 error
		})
	}
	fake.validateCustomMetricTypesReturnsOnCall[i] = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakePolicyDB) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.getAppIdsMutex.RLock()
	defer fake.getAppIdsMutex.RUnlock()
	fake.getAppPolicyMutex.RLock()
	defer fake.getAppPolicyMutex.RUnlock()
	fake.retrievePoliciesMutex.RLock()
	defer fake.retrievePoliciesMutex.RUnlock()
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	fake.emitHealthMetricsMutex.RLock()
	defer fake.emitHealthMetricsMutex.RUnlock()
	fake.getCustomMetricsCredsMutex.RLock()
	defer fake.getCustomMetricsCredsMutex.RUnlock()
	fake.validateCustomMetricsCredsMutex.RLock()
	defer fake.validateCustomMetricsCredsMutex.RUnlock()
	fake.validateCustomMetricTypesMutex.RLock()
	defer fake.validateCustomMetricTypesMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakePolicyDB) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ db.PolicyDB = new(FakePolicyDB)
