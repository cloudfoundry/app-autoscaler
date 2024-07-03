package testhelpers

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	rpc "code.cloudfoundry.org/go-log-cache/v3/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/v10/rpc/loggregator_v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type MockLogCache struct {
	mutex sync.Mutex

	lis net.Listener

	readRequestsCount int
	readResponses     map[string]*rpc.ReadResponse
	readErrors        map[string]error

	instantQueryResults map[string]*rpc.PromQL_InstantQueryResult
	instantQueryErrors  map[string]error

	tlsConfig *tls.Config

	rpc.UnimplementedEgressServer
	rpc.UnimplementedIngressServer
	rpc.UnimplementedPromQLQuerierServer
}

func NewMockLogCache(tlsConfig *tls.Config) *MockLogCache {
	return &MockLogCache{
		tlsConfig: tlsConfig,

		readResponses: map[string]*rpc.ReadResponse{},
		readErrors:    map[string]error{},

		instantQueryResults: map[string]*rpc.PromQL_InstantQueryResult{},
		instantQueryErrors:  map[string]error{},
	}
}

func (m *MockLogCache) Start(port int) error {
	var err error
	m.lis, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}

	var srv *grpc.Server
	if m.tlsConfig == nil {
		srv = grpc.NewServer()
	} else {
		srv = grpc.NewServer(grpc.Creds(credentials.NewTLS(m.tlsConfig)))
	}
	rpc.RegisterIngressServer(srv, m)
	rpc.RegisterEgressServer(srv, m)
	rpc.RegisterPromQLQuerierServer(srv, m)

	//nolint:errcheck
	go srv.Serve(m.lis)

	return nil
}

func (m *MockLogCache) URL() string {
	return m.lis.Addr().String()
}

func (m *MockLogCache) Stop() {
	m.lis.Close()
}

func (m *MockLogCache) Read(ctx context.Context, r *rpc.ReadRequest) (*rpc.ReadResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.readRequestsCount++

	response := &rpc.ReadResponse{Envelopes: &loggregator_v2.EnvelopeBatch{Batch: []*loggregator_v2.Envelope{}}}
	if storedResponse, ok := m.readResponses[r.SourceId]; ok {
		response = storedResponse
	}

	err := errors.New("default-error")
	if storedError, ok := m.readErrors[r.SourceId]; ok {
		err = storedError
	}

	return response, err
}

func (m *MockLogCache) ReadRequestsCount() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.readRequestsCount
}

func (m *MockLogCache) ReadReturns(sourceId string, response *rpc.ReadResponse, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.readResponses[sourceId] = response
	m.readErrors[sourceId] = err
}

func (m *MockLogCache) InstantQuery(ctx context.Context, r *rpc.PromQL_InstantQueryRequest) (*rpc.PromQL_InstantQueryResult, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// every query carries the source_id, example "http{source_id='foo'}"
	// find stored mock-result or mock-error by checking if any of the stored source ids is contained in the query
	query := r.Query
	for sourceId := range m.instantQueryResults {
		if strings.Contains(query, sourceId) {
			return m.instantQueryResults[sourceId], m.instantQueryErrors[sourceId]
		}
	}

	return nil, fmt.Errorf("default-error(query: %s)", query)
}

func (m *MockLogCache) InstantQueryReturns(sourceId string, result *rpc.PromQL_InstantQueryResult, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.instantQueryResults[sourceId] = result
	m.instantQueryErrors[sourceId] = err
}

func NewTLSConfig(caPath, certPath, keyPath, cn string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         cn,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: false,
	}

	caCertBytes, err := os.ReadFile(caPath)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCertBytes); !ok {
		return nil, errors.New("cannot parse ca cert")
	}

	tlsConfig.RootCAs = caCertPool

	return tlsConfig, nil
}
