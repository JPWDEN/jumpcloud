package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jumpcloud/logs"
	"github.com/jumpcloud/service"
	"github.com/jumpcloud/testclient"
)

func main() {
	//Initialize logging system
	infoLog, _, _, errorLog := logs.InitLogger(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	infoLog.Printf("Starting main service")

	//Set environment variables
	port := os.Getenv("PORT")
	addr := os.Getenv("ADDRESS")
	if port == "" || addr == "" {
		port = ":8080"
		addr = "localhost"
	}
	test := false
	if os.Getenv("TEST") == "true" {
		test = true
	}

	//Instantiate server and multiplexer, register endpoints, and start listening
	svc := service.NewServer(infoLog, errorLog)
	mux := http.NewServeMux()
	mux.HandleFunc("/hash", svc.HashPassword)
	mux.HandleFunc("/hash/", svc.CheckPassword)
	mux.HandleFunc("/stats", svc.GetAPIStats)
	mux.HandleFunc("/shutdown", svc.Shutdown)

	go func() {
		infoLog.Printf("Starting API on %s", addr)
		errorLog.Printf("Server error: %v", http.ListenAndServe(port, mux))
	}()

	//Run testclient
	if test {
		client := testclient.NewClient([]string{"angryMonkey"}, infoLog, errorLog)
		infoLog.Printf("Running test client")
		client.RunClient()
	}

	//Block main thread from completing until the correct signal is received
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	stopCall := <-stop
	switch stopCall {
	case syscall.SIGTERM:
		infoLog.Printf("API service shutting down gracefully")
	case syscall.SIGINT:
		infoLog.Printf("API service shutting down gracefully")
	}

	infoLog.Printf("Main service ending")
}
