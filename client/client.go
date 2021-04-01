
package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"
	"errors"
	"net/http"
	"strings"
 


	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
const (
	defaultTimeout = 1*time.Minute
)
const (
	connected = iota
	closed = iota
	retrying = iota
	timedout = iota
)
type clientConn struct {
	wsConn *websocket.Conn
	url string
	headers http.Header
	status int
	connectedChannel chan struct{}

	done chan struct{}
}
func NewClientConn(url string, headers http.Header, interrupted <- chan os.Signal) (*clientConn, error) {
	timer := time.NewTimer(defaultTimeout)
	for {
		select {
		case sig := <-interrupted: return nil, fmt.Errorf("Got OS ingerrupted", sig.String())
		case <-timer.C: return nil, fmt.Errorf("Max timeout reached %s...", defaultTimeout.String())

		default:

			c, _, err := websocket.DefaultDialer.Dial(url, headers)
			if err != nil {
				if !strings.Contains(err.Error(), "connection refused") {
					return nil, err
				}
			} else {
				return &clientConn{wsConn: c, url: url, headers: headers, status: connected, connectedChannel: make(chan struct{}, 10),
					done : make (chan struct{}, 10),
				}, nil
			}
		}
	}

}
func (cc *clientConn) retryConnection () error {
	cc.status = retrying
	start := time.Now()
	for {
		select {
		case <- cc.done:
			return fmt.Errorf("interrupted..")
		default:
			c, _, connErr := websocket.DefaultDialer.Dial(cc.url, cc.headers)
			if connErr != nil {
				// should wait until up
				log.Printf("Still error connecting to %s, connErr=%v\n", cc.url, connErr)
				if time.Since(start) > defaultTimeout {
					cc.status = timedout
					cc.connectedChannel <- struct{}{}
					return  fmt.Errorf("Connection close failed to retry after %v seconds to %s", defaultTimeout.Seconds(), cc.url)
				}
				time.Sleep(5*time.Second)
			} else {
				log.Println("Got connection??? ")
				// otherwise go on
				cc.wsConn = c
				cc.status = connected
				cc.connectedChannel <- struct{}{}
				return nil
			}
		}
	}
}
func (cc *clientConn) readMessageWithRetry() (messageType int, p []byte, err error) {
	if cc.wsConn != nil {
		msgType, message, err := cc.wsConn.ReadMessage()
		if err != nil {
			// see if it's socket close -- the err may have the word close, then we'll try to reopen
			log.Printf("ReadMessage got err = %v", err)
			if strings.Contains(err.Error(), "close") {
				cc.wsConn = nil
				if connErr := cc.retryConnection(); connErr != nil {
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
func (cc *clientConn) WriteMessage(messageType int, data []byte) error {
	if cc.status == retrying {
		<- cc.connectedChannel
	}
	if cc.status != connected {
		return fmt.Errorf("Connection has issue!!! bailed out")
	}
	return cc.writeMessageWithRetry(messageType, data)
}
func (cc *clientConn) writeMessageWithRetry(messageType int, data[]byte) error {
	var err error
	if cc.wsConn != nil {
		err = cc.wsConn.WriteMessage(messageType, data)
		if err == nil {
			return nil
		}
		log.Println("writeMessageWithRetry -- [1] %v", err)
		if strings.Contains(err.Error(), "close") {
			cc.wsConn = nil
			if connErr := cc.retryConnection(); connErr != nil {
				return   connErr
			}
			fmt.Printf("writeMessage with Retry  == ?? ")
			if cc.wsConn != nil {
				return cc.wsConn.WriteMessage(messageType, data)
			} else {
				// ???
				fmt.Printf("what?? ...")
			}
		}
	}
	return fmt.Errorf("connection error!")
}
func (cc *clientConn) close () {
	if cc.wsConn != nil {
		cc.wsConn.Close()
		cc.wsConn = nil
	}
}
func (cc *clientConn) tellServerWeAreClosing() {
	if cc.wsConn != nil {
		if err := cc.wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
			log.Println("closeAsWriter:", err)
			return
		}
	}
}
func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 100)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/echo"}
	log.Printf("connecting to %s", u.String())

	clientConn, err := NewClientConn(u.String(), nil, interrupt)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer clientConn.close()


	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-interrupt:
				log.Println("interrupt in reading")
				 clientConn.done <- struct{}{}

				// Cleanly close the connection by sending a close message and then
				// waiting (with timeout) for the server to close the connection.
				clientConn.tellServerWeAreClosing()
				select {
				case <-time.After(time.Second):
					return
				}
			default:
				_, message, err := clientConn.readMessageWithRetry()
				if err != nil {
					log.Println("read:", err)
					return
				}
				log.Printf("recv: %s", message)
			}

		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	wg.Add(1)
	go func () {
		defer wg.Done()
		for {
			select {
			case <-clientConn.done:
				return
			case t := <-ticker.C:
				log.Println("WANT TO WRITE...")
				err := clientConn.WriteMessage(websocket.TextMessage, []byte("next:"+t.String()))
				if err != nil {
					log.Println("write:", err)
					return
				}
			case <-interrupt:
				log.Println("interrupt in writing")
				clientConn.done <- struct{}{}
				// Cleanly close the connection by sending a close message and then
				// waiting (with timeout) for the server to close the connection.
				clientConn.tellServerWeAreClosing()
				select {
				case <-time.After(time.Second):
					return
				}
			}
		}
	} ()

	wg.Wait()
	log.Println("The END")
}
