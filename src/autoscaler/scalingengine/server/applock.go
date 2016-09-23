package server

import (
	"sync"
)

type AppLock struct {
	applocks map[string]*sync.Mutex
	mutex    *sync.Mutex
}

func NewAppLock() *AppLock {
	return &AppLock{
		applocks: map[string]*sync.Mutex{},
		mutex:    &sync.Mutex{},
	}
}

func (al *AppLock) Lock(appId string) {
	al.mutex.Lock()
	appLock := al.applocks[appId]
	if appLock == nil {
		appLock = &sync.Mutex{}
		al.applocks[appId] = appLock
	}
	al.mutex.Unlock()

	appLock.Lock()
}

func (al *AppLock) UnLock(appId string) {
	al.mutex.Lock()
	appLock := al.applocks[appId]
	al.mutex.Unlock()

	if appLock != nil {
		appLock.Unlock()
	} else {
		panic("unlock app that was not locked")
	}
}
