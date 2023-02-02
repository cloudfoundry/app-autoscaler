package helpers

import (
	"crypto/tls"
	"fmt"
	"net/url"

	"net/http"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"
	//TODO remove static check and use non deprecated version https://github.com/cloudfoundry/app-autoscaler-release/issues/978
	//nolint:staticcheck
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

type WSHelper interface {
	SetupConn() error
	CloseConn() error
	IsClosed() bool
	Write(envelope *loggregator_v2.Envelope) error
	Read() error
	Ping() error
}

type connection struct {
	rwMu sync.RWMutex
	conn *websocket.Conn
}

func (c *connection) getConnection() *websocket.Conn {
	c.rwMu.RLock()
	defer c.rwMu.RUnlock()
	return c.conn
}
func (c *connection) setConnection(conn *websocket.Conn) {
	c.rwMu.Lock()
	defer c.rwMu.Unlock()
	c.conn = conn
}

func (c *connection) Close() error {
	c.rwMu.Lock()
	defer c.rwMu.Unlock()
	err := c.conn.Close()
	c.conn = nil
	return err
}

var _ WSHelper = &WsHelper{}

type WsHelper struct {
	dialer             websocket.Dialer
	maxSetupRetryCount int
	maxCloseRetryCount int
	retryDelay         time.Duration
	logger             lager.Logger
	metricServerURL    string
	connection         connection
	CloseWaitTime      time.Duration
}

func NewWSHelper(metricServerURL string, tlsConfig *tls.Config, handshakeTimeout time.Duration, logger lager.Logger, maxSetupRetryCount int, maxCloseRetryCount int, retryDelay time.Duration) *WsHelper {
	return &WsHelper{
		metricServerURL: metricServerURL,
		dialer: websocket.Dialer{
			TLSClientConfig:  tlsConfig,
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: handshakeTimeout,
		},
		logger:             logger.Session("WSHelper"),
		maxSetupRetryCount: maxSetupRetryCount,
		maxCloseRetryCount: maxCloseRetryCount,
		retryDelay:         retryDelay,
		CloseWaitTime:      5 * time.Second,
	}
}

func (wh *WsHelper) SetupConn() error {
	wh.logger.Info("setup-new-ws-connection")
	URL, err := url.Parse(wh.metricServerURL)
	if err != nil {
		return err
	}

	if URL.Scheme != "wss" && URL.Scheme != "ws" {
		return fmt.Errorf("Invalid scheme '%s'", URL.Scheme)
	}
	retryCount := 0
	for {
		// dial docs says not to close the response body by the application
		//nolint:bodyclose
		con, _, err := wh.dialer.Dial(wh.metricServerURL, nil)
		if err != nil {
			wh.logger.Error("failed-to-create-websocket-connection-to-metricserver", err, lager.Data{"metricServerURL": wh.metricServerURL})
			if retryCount < wh.maxSetupRetryCount {
				retryCount++
				time.Sleep(wh.retryDelay)
			} else {
				return fmt.Errorf("failed after %d retries: %w", retryCount, err)
			}
		} else {
			go func() {
				for {
					_, _, err := con.ReadMessage()
					if err != nil {
						wh.logger.Error("failed-to-read-message", err)
						return
					}
				}
			}()
			wh.connection.setConnection(con)
			return nil
		}
	}
}

func (wh *WsHelper) IsClosed() bool {
	return wh.connection.getConnection() == nil
}

func (wh *WsHelper) CloseConn() error {
	retryCount := 0
	for {
		wh.logger.Info("close-ws-connection")
		err := wh.connection.getConnection().WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Time{})
		if err != nil {
			if retryCount < wh.maxCloseRetryCount {
				retryCount++
				time.Sleep(wh.retryDelay)
			} else {
				return fmt.Errorf("failed to close correctly after %d retries: %w", retryCount, err)
			}
		} else {
			go func() {
				time.AfterFunc(wh.CloseWaitTime, func() {
					_ = wh.connection.Close()
				})
			}()
			return nil
		}
	}
}

func (wh *WsHelper) Write(envelope *loggregator_v2.Envelope) error {
	bytes, err := proto.Marshal(envelope)
	if err != nil {
		wh.logger.Error("failed-to-marshal-envelope", err, lager.Data{"envelope": envelope})
		return err
	}
	wh.logger.Debug("writing-envelope-to-server", lager.Data{"envelope": envelope})
	err = wh.connection.getConnection().WriteMessage(websocket.BinaryMessage, bytes)
	//TODO should retry sending a message.
	if err != nil {
		wh.logger.Error("failed-to-write-envelope", err)
		return wh.reconnect()
	}
	return nil
}
func (wh *WsHelper) Read() error {
	return nil
}
func (wh *WsHelper) Ping() error {
	err := wh.connection.getConnection().WriteControl(websocket.PingMessage, nil, time.Now().Add(1*time.Second))
	if err != nil {
		wh.logger.Error("failed-to-send-ping", err)
		return wh.reconnect()
	}
	return nil
}

func (wh *WsHelper) reconnect() error {
	err := wh.CloseConn()
	if err != nil {
		wh.logger.Error("failed-to-close-websocket-connection", err)
	}

	err = wh.SetupConn()
	if err != nil {
		return err
	}
	return nil
}
