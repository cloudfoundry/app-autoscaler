package cf_test

import (
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
)

type ConnectionWatcher struct {
	maxActive   int32
	connections sync.Map
	wrap        func(net.Conn, http.ConnState)
}

func NewConnectionWatcher(wrapping func(net.Conn, http.ConnState)) *ConnectionWatcher {
	return &ConnectionWatcher{wrap: wrapping}
}

// OnStateChange records open connections in response to connection
// state changes. Set net/http Server.ConnState to this method
// as value.
func (cw *ConnectionWatcher) OnStateChange(conn net.Conn, state http.ConnState) {
	switch state {
	case http.StateClosed, http.StateHijacked:
		cw.Remove(conn)
	default:
		cw.Add(conn, state)
	}
	if cw.wrap != nil {
		cw.wrap(conn, state)
	}
}

// Count returns the number of connections at the time
// the call.
func (cw *ConnectionWatcher) GetStates() map[string]int {
	result := map[string]int{}
	cw.connections.Range(func(key, value any) bool {
		state := value.(http.ConnState).String()
		count, ok := result[state]
		if ok {
			result[state] = count + 1
		} else {
			result[state] = 1
		}
		return true
	})

	return result
}

// Add adds c to the number of active connections.
func (cw *ConnectionWatcher) Add(c net.Conn, state http.ConnState) {
	cw.connections.Store(c, state)
	done := false
	for !done {
		prev := atomic.LoadInt32(&cw.maxActive)
		count := cw.Count()
		if count > prev {
			done = atomic.CompareAndSwapInt32(&cw.maxActive, prev, count)
		} else {
			done = true
		}
	}
}

func (cw *ConnectionWatcher) Remove(c net.Conn) {
	cw.connections.Delete(c)
}

func (cw *ConnectionWatcher) MaxOpenConnections() int32 {
	return atomic.LoadInt32(&cw.maxActive)
}
func (cw *ConnectionWatcher) Count() int32 {
	count := int32(0)
	cw.connections.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}

func (cw *ConnectionWatcher) printStats(title string) {
	GinkgoWriter.Printf("\n# %s\n", title)
	for key, value := range cw.GetStates() {
		GinkgoWriter.Printf("\t%s:\t%d\n", key, value)
	}
}
