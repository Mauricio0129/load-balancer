package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {

	http.HandleFunc("GET /health", health)
	http.HandleFunc("/", handletraffic)

	backends := []string{":9000", ":9001", ":9002", ":9050", ":9051", ":9052"}

	backendsToCancel := make(map[string]context.CancelFunc)

	for _, backend := range backends {

		ctx, cancel := context.WithCancel(context.Background())

		backendsToCancel[backend] = cancel

		go StartServerandStopperThread(backend, ctx)
	}

	readinput(backendsToCancel)
}

func StartServerandStopperThread(addr string, ctx context.Context) {
	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: time.Second * 5,
		WriteTimeout:      time.Second * 5,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		server.Shutdown(shutdownCtx)
	}()

	server.ListenAndServe()
}

func readinput(backendsToCancel map[string]context.CancelFunc) {

	for {

		var w1, w2 string

		fmt.Println("Enter command in this format server address action ON/OFF sample: :8000 OFF")
		_, err := fmt.Scanln(&w1, &w2)

		if err != nil {
			fmt.Printf("Error: %v\n\n/2", err)
			clearInputBuffer()
			continue
		}

		cancelFunc, exists := backendsToCancel[w1]

		if !exists {
			fmt.Printf("Error: Target backend address '%s' not found in active registry!\n\n", w1)
			continue
		}

		cancelFunc()
		fmt.Printf("Successfully triggered graceful shutdown signal for backend %s\n\n", w1)
	}
}

func clearInputBuffer() {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		_ = scanner.Text()
	}
}

func handletraffic(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("Hello from server: %s", r.Host)
	io.WriteString(w, message)
}

func health(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "/pong")
}
