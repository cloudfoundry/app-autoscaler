package testhelpers

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"

	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"google.golang.org/grpc/credentials"
)

type TestIngressServer struct {
	Receivers    chan loggregator_v2.Ingress_BatchSenderServer
	SendReceiver chan *loggregator_v2.EnvelopeBatch
	addr         string
	tlsConfig    *tls.Config
	grpcServer   *grpc.Server
	grpc.ServerStream
	loggregator_v2.UnimplementedIngressServer
}

func NewTestIngressServer(serverCert, serverKey, caCert string) (*TestIngressServer, error) {
	cert, err := tls.LoadX509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		ClientAuth:         tls.RequestClientCert,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}
	caCertBytes, err := os.ReadFile(caCert)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertBytes)
	tlsConfig.RootCAs = caCertPool

	return &TestIngressServer{
		tlsConfig:    tlsConfig,
		Receivers:    make(chan loggregator_v2.Ingress_BatchSenderServer),
		SendReceiver: make(chan *loggregator_v2.EnvelopeBatch, 100),
		addr:         "localhost:0",
	}, nil
}

func (t *TestIngressServer) GetAddr() string {
	return t.addr
}

func (*TestIngressServer) Sender(srv loggregator_v2.Ingress_SenderServer) error {
	return nil
}

func (t *TestIngressServer) BatchSender(srv loggregator_v2.Ingress_BatchSenderServer) error {
	t.Receivers <- srv
	<-srv.Context().Done()

	return nil
}

func (t *TestIngressServer) Send(_ context.Context, b *loggregator_v2.EnvelopeBatch) (*loggregator_v2.SendResponse, error) {
	t.SendReceiver <- b
	return &loggregator_v2.SendResponse{}, nil
}

func (t *TestIngressServer) Start() error {
	listener, err := net.Listen("tcp4", t.addr)
	if err != nil {
		return err
	}
	t.addr = listener.Addr().String()

	var opts []grpc.ServerOption
	if t.tlsConfig != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(t.tlsConfig)))
	}
	t.grpcServer = grpc.NewServer(opts...)

	loggregator_v2.RegisterIngressServer(t.grpcServer, t)

	go func() {
		_ = t.grpcServer.Serve(listener)
	}()

	return nil
}

func (t *TestIngressServer) Stop() {
	t.grpcServer.Stop()
}
