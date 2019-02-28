package testhelpers

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"time"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

type FakeEventProducer struct {
	server             *grpc.Server
	addr               string
	mu                 sync.Mutex
	connectionAttempts int
	actualReq          *loggregator_v2.EgressBatchRequest
	envelops           []*loggregator_v2.Envelope
	certFile           string
	keyFile            string
	caCertFile         string
	emitInterval       time.Duration
}

func NewFakeEventProducer(certFile string, keyFile string, caCertFile string, emitInterval time.Duration) (*FakeEventProducer, error) {
	if emitInterval == 0 {
		emitInterval = 500 * time.Millisecond
	}
	f := &FakeEventProducer{
		certFile:     certFile,
		keyFile:      keyFile,
		caCertFile:   caCertFile,
		emitInterval: emitInterval,
	}

	return f, nil
}

func (f *FakeEventProducer) Receiver(
	*loggregator_v2.EgressRequest,
	loggregator_v2.Egress_ReceiverServer,
) error {

	return grpc.Errorf(codes.Unimplemented, "use BatchedReceiver instead")
}

func (f *FakeEventProducer) BatchedReceiver(
	req *loggregator_v2.EgressBatchRequest,
	srv loggregator_v2.Egress_BatchedReceiverServer,
) error {
	f.mu.Lock()
	f.connectionAttempts++
	f.actualReq = req
	f.mu.Unlock()
	var i int
	for range time.Tick(f.emitInterval) {
		fpEnvs := []*loggregator_v2.Envelope{}
		for _, e := range f.envelops {
			fpEnvs = append(fpEnvs, &loggregator_v2.Envelope{
				SourceId:       e.SourceId,
				Message:        e.Message,
				Tags:           e.Tags,
				DeprecatedTags: e.DeprecatedTags,
				Timestamp:      time.Now().UnixNano(),
			})
		}
		srv.Send(&loggregator_v2.EnvelopeBatch{
			Batch: fpEnvs,
		})
		i++
	}
	return nil
}

func (f *FakeEventProducer) Start() {
	addr := f.addr
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	var lis net.Listener
	for i := 0; ; i++ {
		var err error
		lis, err = net.Listen("tcp", addr)
		if err != nil {
			// This can happen if the port is already in use...
			if i < 50 {
				log.Printf("failed to bind for fake producer. Trying again (%d/50)...: %s", i+1, err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			panic(err)
		}
		break
	}
	f.addr = lis.Addr().String()
	c, err := NewServerMutualTLSConfig(f.certFile, f.keyFile, f.caCertFile)
	if err != nil {
		panic(err)
	}
	opt := grpc.Creds(credentials.NewTLS(c))
	f.server = grpc.NewServer(opt)
	loggregator_v2.RegisterEgressServer(f.server, f)

	go f.listen(lis)
}

func (f *FakeEventProducer) listen(lis net.Listener) {
	_ = f.server.Serve(lis)
}

func (f *FakeEventProducer) Stop() bool {
	if f.server == nil {
		return false
	}

	f.server.Stop()
	f.server = nil
	return true
}

func (f *FakeEventProducer) GetActualReq() *loggregator_v2.EgressBatchRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.actualReq
}

func (f *FakeEventProducer) GetConnectionAttempts() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.connectionAttempts
}
func (f *FakeEventProducer) GetAddr() string {
	return f.addr
}

func (f *FakeEventProducer) SetEnvelops(envelops []*loggregator_v2.Envelope) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.envelops = envelops

}

func NewServerMutualTLSConfig(certFile, keyFile, caCertFile string) (*tls.Config, error) {

	tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load keypair: %s", err)
	}

	certBytes, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read ca cert file: %s", err)
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(certBytes); !ok {
		return nil, errors.New("unable to load ca cert file")
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		Certificates:       []tls.Certificate{tlsCert},
		ClientCAs:          caCertPool,
		ClientAuth:         tls.RequireAndVerifyClientCert,
	}

	return tlsConfig, nil
}

func NewClientMutualTLSConfig(
	certFile string,
	keyFile string,
	caCertFile string,
	serverName string,
) (*tls.Config, error) {
	tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load keypair: %s", err)
	}

	certBytes, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read ca cert file: %s", err)
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(certBytes); !ok {
		return nil, errors.New("unable to load ca cert file")
	}

	certificate, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return nil, err
	}

	verifyOptions := x509.VerifyOptions{
		Roots: caCertPool,
		KeyUsages: []x509.ExtKeyUsage{
			x509.ExtKeyUsageAny,
		},
	}
	if _, err := certificate.Verify(verifyOptions); err != nil {
		return nil, fmt.Errorf("failed to verify certificate: %s", err)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		Certificates:       []tls.Certificate{tlsCert},
		ServerName:         serverName,
		RootCAs:            caCertPool,
	}

	return tlsConfig, err
}
