package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader websocket.Upgrader

type ClientManager struct {
	clients map[*Client]bool
	broadcast chan []byte
	register chan *Client
	unregister chan *Client
}

func newClientManager() *Clientmanager {
	return &Clientmanager{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (cm *ClientManager) run() {
	for {
		select {
		case client := <-cm.register:
			cm.clients[client] = true
		case client := <-cm.unregister:
			if _, ok := cm.clients[client]; ok {
				delete(cm.clients, client)
				close(client.send)
			}
		case message := <-cm.broadcast:
			for client := range cm.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(cm.clients, client)
				}
			}
		}
	}
}

type Client struct {
	manager *ClientManager

	// The websocket connection.
	conn *websocket.Conn
}

func main() {
	s := make(chan os.Signal, 1)
	signal.Notify(s,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGKILL)
	r := mux.NewRouter()

    cm := newClientManager()
    cm.run()

    upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	r.Path("/connect").Methods("GET").HandlerFunc(ConnectWebSocketHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", 6060),
		Handler: r,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Println("Running error")
			fmt.Println(err)
		}
	}()

	sig := <-s
	fmt.Println("Get signal notification.")
	fmt.Println(sig)
	fmt.Println("Start to shutdown server. reset notification user list")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Println("HTTP Server shutdown error")
		fmt.Println(err)
	}
}

// api
func ConnectWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Fail to upgrade %s", err)
		return
	}
	fmt.Println("success to get websocket")

    client := &Client{}

    go client.readPump()
}
