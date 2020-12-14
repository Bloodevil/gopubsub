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

func main() {
	s := make(chan os.Signal, 1)
	signal.Notify(s,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGKILL)
	r := mux.NewRouter()

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
	defer conn.Close()
	if err != nil {
		fmt.Println("Fail to upgrade %s", err)
		return
	}
	fmt.Println("success to get websocket")
}
