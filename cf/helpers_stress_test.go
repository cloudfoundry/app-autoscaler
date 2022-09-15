package cf_test

import (
	"context"
	"errors"
	"net/http/httptrace"
	"sync"
	"sync/atomic"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	. "github.com/onsi/ginkgo/v2"
)

type reqStats struct {
	idleUsed     int32
	reUsed       int32
	numRequests  int32
	numResponses int32
}

func (s *reqStats) Report() {
	GinkgoWriter.Printf("\n# Client trace stats\n")
	GinkgoWriter.Printf("\tidle pool connections used:\t%d\n", atomic.LoadInt32(&s.idleUsed))
	GinkgoWriter.Printf("\tconnections re-used:\t\t%d\n", s.GetReused())
	GinkgoWriter.Printf("\tnumber of requests:\t\t\t%d\n", atomic.LoadInt32(&s.numRequests))
	GinkgoWriter.Printf("\tnumber of responses:\t\t\t%d\n", atomic.LoadInt32(&s.numResponses))
}

func (s reqStats) GetReused() int32 {
	return atomic.LoadInt32(&s.reUsed)
}

type apiCall func(ctx context.Context, client cf.ContextClient) error

func doApiRequest(stats *reqStats, apiCall apiCall) int64 {
	clientTrace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			if info.WasIdle {
				atomic.AddInt32(&stats.idleUsed, 1)
			}
			if info.Reused {
				atomic.AddInt32(&stats.reUsed, 1)
			}
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			atomic.AddInt32(&stats.numRequests, 1)
		},
		GotFirstResponseByte: func() {
			atomic.AddInt32(&stats.numResponses, 1)
		},
	}
	traceCtx := httptrace.WithClientTrace(context.Background(), clientTrace)
	numErr := 0
	var cfError = &cf.CfError{}
	anErr := cfc.Login()
	if anErr != nil && !errors.As(anErr, &cfError) {
		GinkgoWriter.Printf(" Error: %+v\n", anErr)
		numErr += 1
	}
	anErr = apiCall(traceCtx, cfc.GetCtxClient())

	if anErr != nil && !errors.As(anErr, &cfError) {
		GinkgoWriter.Printf(" Error: %+v\n", anErr)
		numErr += 1
	}
	return int64(numErr)
}

func doStressTest(numberConcurrent, numberRequestPerThread int, stats *reqStats, apiCall apiCall) int64 {
	var numErrors int64 = 0
	wg := sync.WaitGroup{}
	for i := 0; i < numberConcurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < numberRequestPerThread; i++ {
				atomic.AddInt64(&numErrors, doApiRequest(stats, apiCall))
			}
		}()
	}
	wg.Wait()
	return numErrors
}
