package main

func main() {
    s := make(chan so.Signal, 1)
    signal.Notify(s,
        syscall.SIGHUP,
        syscall.SIGINT,
        syscall.SIGTERM,
        syscall.SIGQUIT,
        syscall.SIGKILL)
    r := mux.NewRouter()

    r.Path("/connect").Methods("GET").HandlerFunc(H(ConnectWebSocketHandler))

    server := &http.Server{
            Addr:    fmt.Sprintf(":%d", cfg.Port),
            Handler: r,
    }
    go func() {
            if err := server.ListenAndServe(); err != nil {
                    log.Err("Running error", log.Params{"error": err})
            }
    }()

    sig := <-s
    log.Info("Get signal notification.", log.Params{"sig": sig})
    log.Info("Start to shutdown server. reset notification user list", nil)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := server.Shutdown(ctx); err != nil {
            log.Err("HTTP Server shutdown error", log.Params{"err": err})
    }
}
