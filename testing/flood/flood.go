package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	balancerURL := "http://localhost:8080"
	totalRequests := 50000
	concurrencyLimit := 500

	fmt.Println("======= SELECT FLOOD TARGET =======")
	fmt.Println("1) api.localhost Ports 9000-9002")
	fmt.Println("2) www.localhost Ports 9050-9052")
	fmt.Print("Choose target (1 or 2): ")

	var choice string
	fmt.Scanln(&choice)

	var targetHost string
	if choice == "1" || strings.ToLower(choice) == "api" {
		targetHost = "api.localhost:8080"
	} else if choice == "2" || strings.ToLower(choice) == "www" {
		targetHost = "www.localhost:8080"
	} else {
		fmt.Println("Invalid choice. Defaulting to api.localhost")
		targetHost = "api.localhost:8080"
	}

	// shared client — connection reuse across all goroutines
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        concurrencyLimit,
			MaxIdleConnsPerHost: concurrencyLimit,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	fmt.Printf("\nSending traffic to %s (via %s)...\n", targetHost, balancerURL)
	fmt.Printf("Firing %d total requests (%d concurrent)\n\n", totalRequests, concurrencyLimit)

	semaphore := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup

	// atomic counters — no mutex needed for counting
	var successCount int64
	var failCount int64
	results := sync.Map{}

	startTime := time.Now()

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		semaphore <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-semaphore }()

			req, err := http.NewRequest("GET", balancerURL, nil)
			if err != nil {
				atomic.AddInt64(&failCount, 1)
				return
			}
			req.Host = targetHost

			resp, err := client.Do(req)
			if err != nil {
				atomic.AddInt64(&failCount, 1)
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				atomic.AddInt64(&failCount, 1)
				return
			}

			serverID := strings.TrimSpace(string(body))
			atomic.AddInt64(&successCount, 1)

			actual, _ := results.LoadOrStore(serverID, new(int64))
			atomic.AddInt64(actual.(*int64), 1)
		}()
	}

	wg.Wait()
	duration := time.Since(startTime)

	fmt.Println("\n============== FLOOD REPORT ==============")
	fmt.Printf("Target Domain:      %s\n", targetHost)
	fmt.Printf("Total Time Elapsed: %v\n", duration)
	fmt.Printf("Requests/Second:    %.2f\n", float64(totalRequests)/duration.Seconds())
	fmt.Printf("Successful:         %d\n", successCount)
	fmt.Printf("Failed/Timeout:     %d\n\n", failCount)
	fmt.Println("Traffic Distribution Across Cluster:")

	results.Range(func(key, value any) bool {
		count := atomic.LoadInt64(value.(*int64))
		fmt.Printf("  %s -> %d requests\n", key, count)
		return true
	})
	fmt.Println("==========================================")
}
