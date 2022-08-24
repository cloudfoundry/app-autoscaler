package cf_test

import (
	"github.com/orcaman/concurrent-map/v2"
	"net"
	"net/http"
	"sync/atomic"
)

type ConnectionWatcher struct {
	maxActive   int32
	connections cmap.ConcurrentMap[http.ConnState]
	wrap        func(net.Conn, http.ConnState)
}

func NewConnectionWatcher(wrapping func(net.Conn, http.ConnState)) *ConnectionWatcher {
	return &ConnectionWatcher{connections: cmap.New[http.ConnState](), wrap: wrapping}
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
	cw.wrap(conn, state)
}

// Count returns the number of connections at the time
// the call.
func (cw *ConnectionWatcher) GetStates() map[string]int {
	result := map[string]int{}
	for _, value := range cw.connections.Items() {
		state := value.String()
		count, ok := result[state]
		if ok {
			result[state] = count + 1
		} else {
			result[state] = 1
		}
	}

	return result
}

// Add adds c to the number of active connections.
func (cw *ConnectionWatcher) Add(c net.Conn, state http.ConnState) {
	cw.connections.Set(c.LocalAddr().String()+c.RemoteAddr().String(), state)
	done := false
	if !done {
		current := int32(cw.connections.Count())
		prev := atomic.LoadInt32(&cw.maxActive)
		if current > prev {
			done = atomic.CompareAndSwapInt32(&cw.maxActive, prev, current)
		} else {
			done = true
		}
	}
}

func (cw *ConnectionWatcher) Remove(c net.Conn) {
	cw.connections.RemoveCb(c.LocalAddr().String()+c.RemoteAddr().String(), func(key string, v http.ConnState, exists bool) bool { return true })
}

func (cw *ConnectionWatcher) MaxOpenConnections() int32 {
	return atomic.LoadInt32(&cw.maxActive)
}
