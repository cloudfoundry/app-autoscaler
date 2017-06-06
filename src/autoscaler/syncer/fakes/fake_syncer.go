// This file was generated by counterfeiter
package fakes

import (
	"autoscaler/syncer"
	"sync"
)

type FakeSyncer struct {
	SynchronizeStub        func() error
	synchronizeMutex       sync.RWMutex
	synchronizeArgsForCall []struct{}
	synchronizeReturns     struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeSyncer) Synchronize() error {
	fake.synchronizeMutex.Lock()
	fake.synchronizeArgsForCall = append(fake.synchronizeArgsForCall, struct{}{})
	fake.recordInvocation("Synchronize", []interface{}{})
	fake.synchronizeMutex.Unlock()
	if fake.SynchronizeStub != nil {
		return fake.SynchronizeStub()
	}
	return fake.synchronizeReturns.result1
}

func (fake *FakeSyncer) SynchronizeCallCount() int {
	fake.synchronizeMutex.RLock()
	defer fake.synchronizeMutex.RUnlock()
	return len(fake.synchronizeArgsForCall)
}

func (fake *FakeSyncer) SynchronizeReturns(result1 error) {
	fake.SynchronizeStub = nil
	fake.synchronizeReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeSyncer) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.synchronizeMutex.RLock()
	defer fake.synchronizeMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeSyncer) recordInvocation(key string, args []interface{}) {
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

var _ syncer.Syncer = new(FakeSyncer)
