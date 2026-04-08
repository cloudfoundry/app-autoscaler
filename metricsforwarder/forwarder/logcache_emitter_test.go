package forwarder_test

import (
	"context"
	"fmt"
	"net"
	"sync"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/config"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/metricsforwarder/forwarder"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	rpc "code.cloudfoundry.org/go-log-cache/v3/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"google.golang.org/grpc"
)

type fakeIngressServer struct {
	rpc.UnimplementedIngressServer
	mu        sync.Mutex
	envelopes []*loggregator_v2.Envelope
	sendErr   error
}

func (f *fakeIngressServer) Send(_ context.Context, req *rpc.SendRequest) (*rpc.SendResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sendErr != nil {
		return nil, f.sendErr
	}
	f.envelopes = append(f.envelopes, req.GetEnvelopes().GetBatch()...)
	return &rpc.SendResponse{}, nil
}

func (f *fakeIngressServer) ReceivedEnvelopes() []*loggregator_v2.Envelope {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.envelopes
}

var _ = Describe("LogCacheEmitter", func() {
	var (
		listener    net.Listener
		grpcServer  *grpc.Server
		fakeIngress *fakeIngressServer
		emitter     forwarder.MetricForwarder
		err         error
		conf        *config.Config
		logger      *lagertest.TestLogger
		buffer      *gbytes.Buffer
	)

	BeforeEach(func() {
		fakeIngress = &fakeIngressServer{}

		listener, err = net.Listen("tcp", "127.0.0.1:0")
		Expect(err).ToNot(HaveOccurred())

		grpcServer = grpc.NewServer()
		rpc.RegisterIngressServer(grpcServer, fakeIngress)
		go grpcServer.Serve(listener) //nolint:errcheck
	})

	AfterEach(func() {
		grpcServer.Stop()
	})

	JustBeforeEach(func() {
		conf = &config.Config{
			LogCacheConfig: config.LogCacheConfig{
				Address: listener.Addr().String(),
			},
		}

		logger = lagertest.NewTestLogger("logcache-emitter-test")
		buffer = logger.Buffer()
		emitter, err = forwarder.NewLogCacheEmitter(logger, conf)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("NewLogCacheEmitter", func() {
		It("creates emitter without TLS", func() {
			Expect(emitter).ToNot(BeNil())
			Eventually(buffer).Should(gbytes.Say("using-logcache-grpc-insecure"))
		})
	})

	Describe("EmitMetric", func() {
		var metric *models.CustomMetric

		BeforeEach(func() {
			metric = &models.CustomMetric{
				Name:          "queuelength",
				Value:         12,
				Unit:          "bytes",
				InstanceIndex: 123,
				AppGUID:       "dummy-guid",
			}
		})

		It("sends envelope to logcache via gRPC", func() {
			emitter.EmitMetric(metric)

			Eventually(fakeIngress.ReceivedEnvelopes).Should(HaveLen(1))
			received := fakeIngress.ReceivedEnvelopes()[0]
			expected := forwarder.EnvelopeForMetric(metric)
			Expect(received.SourceId).To(Equal(expected.SourceId))
			Expect(received.InstanceId).To(Equal(expected.InstanceId))
			Expect(received.Message).To(Equal(expected.Message))
		})

		When("gRPC send fails", func() {
			BeforeEach(func() {
				fakeIngress.sendErr = fmt.Errorf("send error")
			})

			It("logs the error", func() {
				emitter.EmitMetric(metric)
				Eventually(buffer).Should(gbytes.Say("failed-to-send-metric-to-logcache"))
			})
		})
	})
})
