package synchronization

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/DCMMC/chainlink/core/logger"
	"github.com/DCMMC/chainlink/core/service"
	"github.com/DCMMC/chainlink/core/static"
	"github.com/DCMMC/chainlink/core/utils"

	"github.com/gorilla/websocket"
	"go.uber.org/atomic"
)

var (
	// ErrReceiveTimeout is returned when no message is received after a
	// specified duration in Receive
	ErrReceiveTimeout = errors.New("timeout waiting for message")
)

type ConnectionStatus string

const (
	// ConnectionStatusDisconnected is the default state
	ConnectionStatusDisconnected = ConnectionStatus("disconnected")
	// ConnectionStatusConnected is used when the client is successfully connected
	ConnectionStatusConnected = ConnectionStatus("connected")
	// ConnectionStatusError is used when there is an error
	ConnectionStatusError = ConnectionStatus("error")
)

// SendBufferSize is the number of messages to keep in the buffer before dropping additional ones
const SendBufferSize = 100

const (
	ExplorerTextMessage   = websocket.TextMessage
	ExplorerBinaryMessage = websocket.BinaryMessage
)

// ExplorerClient encapsulates all the functionality needed to
// push run information to explorer.
type ExplorerClient interface {
	service.Service
	Url() url.URL
	Status() ConnectionStatus
	Send(context.Context, []byte, ...int)
	Receive(context.Context, ...time.Duration) ([]byte, error)
}

type NoopExplorerClient struct{}

func (NoopExplorerClient) Url() url.URL                                              { return url.URL{} }
func (NoopExplorerClient) Status() ConnectionStatus                                  { return ConnectionStatusDisconnected }
func (NoopExplorerClient) Start() error                                              { return nil }
func (NoopExplorerClient) Close() error                                              { return nil }
func (NoopExplorerClient) Healthy() error                                            { return nil }
func (NoopExplorerClient) Ready() error                                              { return nil }
func (NoopExplorerClient) Send(context.Context, []byte, ...int)                      {}
func (NoopExplorerClient) Receive(context.Context, ...time.Duration) ([]byte, error) { return nil, nil }

type explorerClient struct {
	utils.StartStopOnce
	conn             *websocket.Conn
	sendText         chan []byte
	sendBinary       chan []byte
	dropMessageCount atomic.Uint32
	receive          chan []byte
	sleeper          utils.Sleeper
	status           ConnectionStatus
	url              *url.URL
	accessKey        string
	secret           string
	logging          bool

	done          chan struct{}
	writePumpDone chan struct{}

	statusMtx sync.RWMutex
}

// NewExplorerClient returns a stats pusher using a websocket for
// delivery.
func NewExplorerClient(url *url.URL, accessKey, secret string, loggingArgs ...bool) ExplorerClient {
	logging := false
	if len(loggingArgs) > 0 {
		logging = loggingArgs[0]
	}
	return &explorerClient{
		url:       url,
		receive:   make(chan []byte),
		sleeper:   utils.NewBackoffSleeper(),
		status:    ConnectionStatusDisconnected,
		accessKey: accessKey,
		secret:    secret,
		logging:   logging,

		sendText:   make(chan []byte, SendBufferSize),
		sendBinary: make(chan []byte, SendBufferSize),
	}
}

// Url returns the URL the client was initialized with
func (ec *explorerClient) Url() url.URL {
	return *ec.url
}

// Status returns the current connection status
func (ec *explorerClient) Status() ConnectionStatus {
	ec.statusMtx.RLock()
	defer ec.statusMtx.RUnlock()
	return ec.status
}

// Start starts a write pump over a websocket.
func (ec *explorerClient) Start() error {
	return ec.StartOnce("Explorer client", func() error {
		ec.done = make(chan struct{})
		go ec.connectAndWritePump()
		return nil
	})
}

// Send sends data asynchronously across the websocket if it's open, or
// holds it in a small buffer until connection, throwing away messages
// once buffer is full.
// func (ec *explorerClient) Receive(durationParams ...time.Duration) ([]byte, error) {
func (ec *explorerClient) Send(ctx context.Context, data []byte, messageTypes ...int) {
	messageType := ExplorerTextMessage
	if len(messageTypes) > 0 {
		messageType = messageTypes[0]
	}
	var send chan []byte
	switch messageType {
	case ExplorerTextMessage:
		send = ec.sendText
	case ExplorerBinaryMessage:
		send = ec.sendBinary
	default:
		log.Panicf("send on explorer client received unsupported message type %d", messageType)
	}
	select {
	case send <- data:
		ec.dropMessageCount.Store(0)
	case <-ctx.Done():
		return
	default:
		ec.logBufferFullWithExpBackoff(data)
	}
}

// logBufferFullWithExpBackoff logs messages at
// 1
// 2
// 4
// 8
// 16
// 32
// 64
// 100
// 200
// 300
// etc...
func (ec *explorerClient) logBufferFullWithExpBackoff(data []byte) {
	count := ec.dropMessageCount.Inc()
	if count > 0 && (count%100 == 0 || count&(count-1) == 0) {
		logger.Warnw("explorer client buffer full, dropping message", "data", data, "droppedCount", count)
	}
}

// Receive blocks the caller while waiting for a response from the server,
// returning the raw response bytes
func (ec *explorerClient) Receive(ctx context.Context, durationParams ...time.Duration) ([]byte, error) {
	duration := defaultReceiveTimeout
	if len(durationParams) > 0 {
		duration = durationParams[0]
	}

	select {
	case data := <-ec.receive:
		return data, nil
	case <-time.After(duration):
		return nil, ErrReceiveTimeout
	case <-ctx.Done():
		return nil, nil
	}
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	// defaultReceiveTimeout is the default amount of time to wait for receipt of messages
	defaultReceiveTimeout = 30 * time.Second
)

// Inspired by https://github.com/gorilla/websocket/blob/master/examples/chat/client.go
// lexical confinement of done chan allows multiple connectAndWritePump routines
// to clean up independent of itself by reducing shared state. i.e. a passed done, not ec.done.
func (ec *explorerClient) connectAndWritePump() {
	for {
		select {
		case <-time.After(ec.sleeper.After()):
			ctx, cancel := utils.ContextFromChan(ec.done)

			logger.Infow("Connecting to explorer", "url", ec.url)
			err := ec.connect(ctx)
			if ctx.Err() != nil {
				cancel()
				return
			} else if err != nil {
				ec.setStatus(ConnectionStatusError)
				logger.Warn("Failed to connect to explorer (", ec.url.String(), "): ", err)
				cancel()
				break
			}
			cancel()

			ec.setStatus(ConnectionStatusConnected)

			logger.Infow("Connected to explorer", "url", ec.url)
			ec.sleeper.Reset()
			ec.writePumpDone = make(chan struct{})
			go ec.readPump()
			ec.writePump()

		case <-ec.done:
			return
		}
	}
}

func (ec *explorerClient) setStatus(s ConnectionStatus) {
	ec.statusMtx.Lock()
	defer ec.statusMtx.Unlock()
	ec.status = s
}

// Inspired by https://github.com/gorilla/websocket/blob/master/examples/chat/client.go#L82
func (ec *explorerClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		ec.wrapConnErrorIf(ec.conn.Close()) // exclusive responsibility to close ws conn
	}()

	for {
		select {
		case message, open := <-ec.sendText:
			if !open {
				ec.wrapConnErrorIf(ec.conn.WriteMessage(websocket.CloseMessage, []byte{}))
			}

			err := ec.writeMessage(message, websocket.TextMessage)
			if err != nil {
				logger.Warnw("websocketStatsPusher: error writing text message", "err", err)
				return
			}

		case message, open := <-ec.sendBinary:
			if !open {
				ec.wrapConnErrorIf(ec.conn.WriteMessage(websocket.CloseMessage, []byte{}))
			}

			err := ec.writeMessage(message, websocket.BinaryMessage)
			if err != nil {
				logger.Warnw("websocketStatsPusher: error writing binary message", "err", err)
				return
			}

		case <-ticker.C:
			ec.wrapConnErrorIf(ec.conn.SetWriteDeadline(time.Now().Add(writeWait)))
			if err := ec.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				ec.wrapConnErrorIf(err)
				return
			}

		case <-ec.writePumpDone:
			return
		case <-ec.done:
			return
		}
	}
}

func (ec *explorerClient) writeMessage(message []byte, messageType int) error {
	ec.wrapConnErrorIf(ec.conn.SetWriteDeadline(time.Now().Add(writeWait)))
	writer, err := ec.conn.NextWriter(messageType)
	if err != nil {
		return err
	}

	if _, err := writer.Write(message); err != nil {
		return err
	}
	if ec.logging {
		logger.Debugw("websocketStatsPusher successfully wrote message", "messageType", messageType, "message", message)
	}

	return writer.Close()
}

func (ec *explorerClient) connect(ctx context.Context) error {
	authHeader := http.Header{}
	authHeader.Add("X-Explore-Chainlink-Accesskey", ec.accessKey)
	authHeader.Add("X-Explore-Chainlink-Secret", ec.secret)
	authHeader.Add("X-Explore-Chainlink-Core-Version", static.Version)
	authHeader.Add("X-Explore-Chainlink-Core-Sha", static.Sha)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, ec.url.String(), authHeader)
	if ctx.Err() != nil {
		return fmt.Errorf("websocketStatsPusher#connect context canceled: %w", ctx.Err())
	} else if err != nil {
		return fmt.Errorf("websocketStatsPusher#connect: %v", err)
	}

	ec.conn = conn
	return nil
}

var expectedCloseMessages = []int{websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure}

const CloseTimeout = 100 * time.Millisecond

// readPump listens on the websocket connection for control messages and
// response messages (text)
//
// For more details on how disconnection messages are handled, see:
//  * https://stackoverflow.com/a/48181794/639773
//  * https://github.com/gorilla/websocket/blob/master/examples/chat/client.go#L56
func (ec *explorerClient) readPump() {
	ec.conn.SetReadLimit(maxMessageSize)
	_ = ec.conn.SetReadDeadline(time.Now().Add(pongWait))
	ec.conn.SetPongHandler(func(string) error {
		_ = ec.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		messageType, message, err := ec.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, expectedCloseMessages...) {
				logger.Warnw("Unexpected close error on ExplorerClient", "err", err)
			}
			close(ec.writePumpDone)
			return
		}

		switch messageType {
		case websocket.TextMessage:
			ec.receive <- message
		}
	}
}

func (ec *explorerClient) wrapConnErrorIf(err error) {
	if err != nil && websocket.IsUnexpectedCloseError(err, expectedCloseMessages...) {
		ec.setStatus(ConnectionStatusError)
		logger.Error(fmt.Sprintf("websocketStatsPusher: %v", err))
	}
}

func (ec *explorerClient) Close() error {
	return ec.StopOnce("Explorer client", func() error {
		close(ec.done)
		return nil
	})
}
