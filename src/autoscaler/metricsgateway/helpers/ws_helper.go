package helpers

import (
	"crypto/tls"
	"fmt"
	"net/url"

	"code.cloudfoundry.org/lager"
	"net/http"
	"sync"
	"time"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

type WSHelper interface {
	SetupConn() error
	CloseConn() error
	Write(envelope *loggregator_v2.Envelope) error
	Read() error
	Ping() error
}

type wshelper struct {
	lock               sync.Mutex
	dialer             websocket.Dialer
	maxSetupRetryCount int
	maxCloseRetryCount int
	retryDelay         time.Duration
	logger             lager.Logger
	metricServerURL    string
	wsConn             *websocket.Conn
}

func NewWSHelper(metricServerURL string, tlsConfig *tls.Config, handshakeTimeout time.Duration, logger lager.Logger, maxSetupRetryCount int, maxCloseRetryCount int, retryDelay time.Duration) WSHelper {
	return &wshelper{
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
	}
}

func (wh *wshelper) SetupConn() error {
	wh.logger.Info("setup-new-ws-connection")
	URL, err := url.Parse(wh.metricServerURL)
	if err != nil {
		return err
	}

	if URL.Scheme != "wss" && URL.Scheme != "ws" {
		return fmt.Errorf("Invalid scheme '%s'", URL.Scheme)
	}
	retryCount := 1
	for {
		con, _, err := wh.dialer.Dial(wh.metricServerURL, nil)
		if err != nil {
			wh.logger.Error("failed-to-create-websocket-connection-to-metricserver", err, lager.Data{"metricServerURL": wh.metricServerURL})
			if retryCount <= wh.maxSetupRetryCount {
				retryCount++
				time.Sleep(wh.retryDelay)
			} else {
				wh.logger.Error("maximum-number-of-setup-retries-reached", err, lager.Data{"maxSetupRetryCount": wh.maxSetupRetryCount})
				return err
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
			wh.lock.Lock()
			defer wh.lock.Unlock()
			wh.wsConn = con
			return nil
		}

	}

}
func (wh *wshelper) CloseConn() error {
	retryCount := 1
	for {
		wh.logger.Info("close-ws-connection")
		err := wh.wsConn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Time{})
		if err != nil {
			wh.logger.Error("failed-to-send-close-message-to-metricserver", err, lager.Data{"current": retryCount})
			if retryCount <= wh.maxCloseRetryCount {
				retryCount++
				wh.logger.Info("retry", lager.Data{"RETRY": retryCount})
				time.Sleep(wh.retryDelay)
			} else {
				wh.logger.Error("maximum-number-of-close-retries-reached", err, lager.Data{"maxCloseRetryCount": wh.maxCloseRetryCount})
				return err
			}
		} else {
			go func() {
				wh.lock.Lock()
				con := wh.wsConn
				wh.lock.Unlock()
				time.AfterFunc(5*time.Second, func() {
					err := con.Close()
					if err != nil {
						wh.logger.Error("failed-to-close-ws-connection", err)
					} else {
						wh.logger.Info("successfully-close-ws-connection")
					}
				})

			}()
			return nil
		}
	}

}
func (wh *wshelper) Write(envelope *loggregator_v2.Envelope) error {
	bytes, err := proto.Marshal(envelope)
	if err != nil {
		wh.logger.Error("failed-to-marshal-envelope", err, lager.Data{"envelope": envelope})
		return err
	}
	wh.logger.Debug("writing-envelope-to-server", lager.Data{"envelope": envelope})
	err = wh.wsConn.WriteMessage(websocket.BinaryMessage, bytes)
	if err != nil {
		wh.logger.Error("failed-to-write-envelope", err)
		return wh.reconnect()
	}
	return nil
}
func (wh *wshelper) Read() error {
	return nil
}
func (wh *wshelper) Ping() error {
	wh.logger.Debug("send-ping")
	err := wh.wsConn.WriteControl(websocket.PingMessage, nil, time.Now().Add(1*time.Second))
	if err != nil {
		wh.logger.Error("failed-to-send-ping", err)
		return wh.reconnect()
	}
	return nil
}

func (wh *wshelper) reconnect() error {
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
