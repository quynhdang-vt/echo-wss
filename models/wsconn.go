package models

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout = 1 * time.Minute
)

type wsConnWrapper interface {
	ReadMessage(ctx context.Context) (messageType int, p []byte, err error)
	WriteMessage(ctx context.Context, messageType int, data []byte) error
	Done() <-chan struct{}
	Close() error
}

const (
	connected = iota
	closed    = iota
	retrying  = iota
	timedout  = iota
)

type clientConn struct {
	wsConn           *websocket.Conn
	url              string
	headers          http.Header
	status           int
	connectedChannel chan struct{}

	done chan struct{}
	maxTimeout       time.Duration
}

func NewClientConn(ctx context.Context, url string, headers http.Header, maxTimeout time.Duration) (wsConnWrapper, error) {
	if maxTimeout == 0 {
		maxTimeout = defaultTimeout
	}
	timer := time.NewTimer(maxTimeout)
	for {
		select {
		case <- ctx.Done():
			return nil, fmt.Errorf("context canceled [NewClientConn]")
		case <-timer.C:
			return nil, fmt.Errorf("Max timeout reached %s...", maxTimeout.String())
		default:
			c, _, err := websocket.DefaultDialer.Dial(url, headers)
			if err != nil {
				if !strings.Contains(err.Error(), "connection refused") {
					return nil, err
				}
			} else {
				return &clientConn{wsConn: c, url: url, headers: headers, status: connected, connectedChannel: make(chan struct{}, 10),
					done: make(chan struct{}, 10),
					maxTimeout: maxTimeout,
				}, nil
			}
		}
	}
}

func (cc *clientConn) retryConnection(ctx context.Context, loc string) error {
	method := fmt.Sprintf("[retryConnection:%s]", loc)
	cc.status = retrying
	start := time.Now()
	for {
		select {
		case <- ctx.Done():
			cc.connectedChannel <- struct{}{}
			return fmt.Errorf("%s context canceled ", method)
		case <-cc.done:
			cc.connectedChannel <- struct{}{}
			return fmt.Errorf("%s interrupted..", method)
		default:
			c, _, connErr := websocket.DefaultDialer.Dial(cc.url, cc.headers)
			if connErr != nil {
				// should wait until up
				log.Printf("%s Still error connecting to %s, connErr=%v\n", method, cc.url, connErr)
				if time.Since(start) > cc.maxTimeout {
					cc.status = timedout
					cc.connectedChannel <- struct{}{}
					return fmt.Errorf("%s -  failed to retry to connect to %s after %v seconds", method, cc.url, cc.maxTimeout.Seconds())
				}
				time.Sleep(1 * time.Second)
			} else {
				log.Println(method, "Got connection??? ")
				// otherwise go on
				cc.wsConn = c
				cc.status = connected
				cc.connectedChannel <- struct{}{}
				return nil
			}
		}
	}
}
func (cc *clientConn) ReadMessage(ctx context.Context) (messageType int, p []byte, err error) {
	const method = "ReadMessage"
	for {
		select {
		case <-ctx.Done():
			return 0, nil, fmt.Errorf("context canceled[ReadMessage]")
		default:
			if cc.wsConn != nil {
				msgType, message, err := cc.wsConn.ReadMessage()
				if err != nil {
					// see if it's socket close -- the err may have the word close, then we'll try to reopen
					if cc.wsConn == nil {
						// some one close this
						if strings.Contains(err.Error(), "closed network connection") {
							// got it.. just return
							return 0, nil, io.EOF
						}
					}
					log.Printf("ReadMessage got err = %v", err)
					if strings.Contains(err.Error(), "close") {
						cc.wsConn = nil
						if connErr := cc.retryConnection(ctx, method); connErr != nil {
							return 0, nil, connErr
						}
						fmt.Printf("readMessageWithRetryConn == ?? ")
						if cc.wsConn != nil {
							return cc.wsConn.ReadMessage()
						} else {
							// ???
							fmt.Printf("what?? ...")
						}
					}

				}
				return msgType, message, err
			}
			return 0, nil, errors.New("reading from a close connection...")
		}
	}
}
func (cc *clientConn) WriteMessage(ctx context.Context, messageType int, data []byte) error {
	const method = "WriteMessage"
	if cc.status == retrying {
		<-cc.connectedChannel
	}
	if cc.status != connected {
		return fmt.Errorf("Connection has issue!!! bailed out, status=%s", cc.status)
	}
	if cc.wsConn != nil {
		err := cc.wsConn.WriteMessage(messageType, data)
		if err == nil {
			return nil
		}
		log.Println(method, " -- [1]", err)
		if strings.Contains(err.Error(), "close") {
			cc.wsConn = nil
			if connErr := cc.retryConnection(ctx, method); connErr != nil {
				return connErr
			}
			if cc.wsConn != nil {
				return cc.wsConn.WriteMessage(messageType, data)
			}
		}
	}
	return fmt.Errorf("connection error!")
}
func (cc *clientConn) Close() error {
	cc.done <- struct{}{}
	cc.status = closed
	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	return cc.tellServerWeAreClosing()
}
func (cc *clientConn) tellServerWeAreClosing() error {
	if cc.wsConn != nil {
		if err := cc.wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
			log.Println("tellServerWeAreClosing:", err)
			return err
		}
		cc.wsConn.Close()
		cc.wsConn = nil
	}
	return nil
}
func (cc *clientConn) Done() <-chan struct{} {
	return cc.done
}
