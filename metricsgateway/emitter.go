package metricsgateway

import (
	"crypto/tls"
	"net/http"
	"sync"
	"time"

	"autoscaler/routes"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

type Emitter interface {
	Accept(envelope *loggregator_v2.Envelope)
	Emit(*loggregator_v2.Envelope) error
}

type EnvelopeEmitter struct {
	logger                     lager.Logger
	metricsServerAddress       string
	handshakeTimeout           time.Duration
	envelopChan                chan *loggregator_v2.Envelope
	bufferSize                 int64
	doneChan                   chan bool
	wsConn                     *websocket.Conn
	lock                       sync.Mutex
	dialer                     websocket.Dialer
	verifyWSConnectionInterval time.Duration
	eclock                     clock.Clock
	ticker                     clock.Ticker
}

func NewEnvelopeEmitter(logger lager.Logger, bufferSize int64, metricsServerAddress string, tlsConfig *tls.Config, handshakeTimeout time.Duration, eclock clock.Clock, verifyWSConnectionInterval time.Duration) *EnvelopeEmitter {
	return &EnvelopeEmitter{
		logger:               logger.Session("EnvelopeEmitter"),
		metricsServerAddress: metricsServerAddress,
		handshakeTimeout:     handshakeTimeout,
		envelopChan:          make(chan *loggregator_v2.Envelope, bufferSize),
		doneChan:             make(chan bool),
		dialer: websocket.Dialer{
			TLSClientConfig:  tlsConfig,
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: handshakeTimeout,
		},
		eclock:                     eclock,
		verifyWSConnectionInterval: verifyWSConnectionInterval,
	}
}
func (e *EnvelopeEmitter) Start() {
	err := e.setupWSConn()
	if err != nil {
		e.logger.Error("failed-to-start-emimtter", err)
		return
	}
	go e.startEmitEnvelope()
	e.logger.Info("started")
}

func (e *EnvelopeEmitter) startEmitEnvelope() {
	e.ticker = e.eclock.NewTicker(e.verifyWSConnectionInterval)
	for {
		select {
		case <-e.doneChan:
			e.logger.Info("stopped")
			return
		case envelope := <-e.envelopChan:
			err := e.Emit(envelope)
			if err != nil {
				e.logger.Error("failed-to-emit-envelope", err, lager.Data{"message": envelope})
			}
		case <-e.ticker.C():
			e.verifyWSConnection()
		}
	}
}

func (e *EnvelopeEmitter) Stop() {
	e.closeWSConn()
	e.doneChan <- true

}

func (e *EnvelopeEmitter) Accept(envelope *loggregator_v2.Envelope) {
	e.logger.Debug("accept-envelope", lager.Data{"envelope": envelope})
	e.envelopChan <- envelope
}
func (e *EnvelopeEmitter) Emit(envelope *loggregator_v2.Envelope) error {
	e.logger.Debug("emit-envelope", lager.Data{"envelope": envelope})
	bytes, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}
	err = e.wsConn.WriteMessage(websocket.BinaryMessage, bytes)
	return err
}

func (e *EnvelopeEmitter) setupWSConn() error {
	e.logger.Info("setup-new-ws-connection")
	con, _, err := e.dialer.Dial(e.metricsServerAddress+routes.EnvelopePath, nil)
	if err != nil {
		e.logger.Error("failed-to-create-websocket-connection-to-metricserver", err, lager.Data{"metricServerUrl": (e.metricsServerAddress + routes.EnvelopePath)})
		return err
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	e.wsConn = con
	return nil
}
func (e *EnvelopeEmitter) closeWSConn() {
	e.logger.Info("close-ws-connection")
	err := e.wsConn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Time{})
	if err != nil {
		e.logger.Error("failed-to-send-close-message-to-metricserver", err)
	}
	go func() {
		e.lock.Lock()
		con := e.wsConn
		e.lock.Unlock()
		timeChan := time.After(5 * time.Second)
		select {
		case <-timeChan:
			err = con.Close()
			if err != nil {
				e.logger.Error("failed-to-close-ws-connection", err)
			}
		}

	}()

}

func (e *EnvelopeEmitter) verifyWSConnection() {
	err := e.wsConn.WriteControl(websocket.PingMessage, nil, time.Time{})
	mt, message, err := e.wsConn.ReadMessage()
	if err != nil {
		e.logger.Error("remote-ws-connection-has-been-closed", err, lager.Data{"messageType": mt, "message": message})
		e.closeWSConn()
		e.setupWSConn()
	}

}
