package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/databricks/cli/libs/testserver"
)

func main() {
	port := flag.Int("port", 0, "Port to listen on (0 for random)")
	flag.Parse()

	// Create listener
	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port
	serverURL := fmt.Sprintf("http://127.0.0.1:%d", actualPort)

	// Create server using libs/testserver
	server := testserver.NewStandalone(serverURL)
	testserver.AddDefaultHandlers(server)

	// Add reset state endpoint for fuzzer
	server.Handle("POST", "/testserver-reset-state", func(req testserver.Request) any {
		server.ResetState()
		return map[string]string{"status": "ok"}
	})

	// Print URL to stdout (fuzzer reads this)
	os.Stdout.WriteString(serverURL + "\n")

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down...")
		listener.Close()
		os.Exit(0)
	}()

	log.Printf("Test server listening on %s", serverURL)

	if err := http.Serve(listener, server.Router); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
