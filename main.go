package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jumpcloud/logs"
	"github.com/jumpcloud/service"
)

func main() {
	infoLog, _, _, errorLog := logs.InitLogger(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	infoLog.Printf("Starting main service")

	port := os.Getenv("PORT")
	addr := os.Getenv("ADDRESS")
	if port == "" || addr == "" {
		port = ":8080"
		addr = "localhost"
	}

	svc := service.NewServer()
	mux := http.NewServeMux()
	mux.HandleFunc("/hash", svc.HashPassword)
	mux.HandleFunc("/hash/", svc.CheckPassword)
	mux.HandleFunc("/stats", svc.GetAPIStats)
	mux.HandleFunc("/shutdown", svc.Shutdown)

	go func() {
		infoLog.Printf("Starting API on %s", addr)
		errorLog.Printf("Server error: %v", http.ListenAndServe(port, mux))
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	infoLog.Printf("Main service ending")
}
