package main

import (
	"context"
	"flag"
	"github.com/quynhdang-vt/echo-wss/models"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 100)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/echo"}
	log.Printf("connecting to %s", u.String())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wsConn, err := models.NewClientConn(ctx, u.String(), nil, 0)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer wsConn.Close()

	timer := time.NewTimer(5*time.Minute)
	defer timer.Stop()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <- timer.C:
				log.Println("timer is up in reading loop")
				cancel()
			case <- ctx.Done():
				wsConn.Close()
				return
			case <-wsConn.Done():
				return
			case <-interrupt:
				log.Println("interrupt in reading loop")
				cancel()
			default:
				_, message, err := wsConn.ReadMessage(ctx)
				if err != nil {
					log.Printf("reading loop gets %v", err)
					return
				}
				log.Printf("recv: %s", message)
			}

		}
	}()

	// write a message ever second
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <- timer.C:
				log.Println("timer is up in writing loop")
				cancel()
			case <-ctx.Done():
				wsConn.Close()
				return
			case <-wsConn.Done():
				return
			case <-interrupt:
				log.Println("interrupt in writing loop")
				cancel()
			case t := <-ticker.C:
				log.Println("WANT TO WRITE...")
				err := wsConn.WriteMessage(ctx, websocket.TextMessage, []byte("client:"+t.String()))
				if err != nil {
					log.Println("write:", err)
					return
				}
			}
		}
	}()

	time.Sleep(90*time.Second) //*time.Minute)
	log.Println("----- in MAIN every one shutdown..")
	cancel()
	log.Println("----- in MAIN 2..")

	wg.Wait()
	log.Println("The END")
}
