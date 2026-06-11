package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

func main() {
	balancerURL := "http://localhost:8080"
	totalRequests := 300
	concurrencyLimit := 15

	// 1. Choose your target domain interactively
	fmt.Println("======= SELECT FLOOD TARGET =======")
	fmt.Println("1) api.localhost (Ports 9000-9002)")
	fmt.Println("2) www.localhost (Ports 9050-9052)")
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
		targetHost = "api.localhost"
	}

	fmt.Printf("\n Sending trafic to %s (via %s)...\n", targetHost, balancerURL)
	fmt.Printf("Firing %d total requests (Max %d concurrent threads)\n\n", totalRequests, concurrencyLimit)

	semaphore := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup

	var mu sync.Mutex
	results := make(map[string]int)

	startTime := time.Now()

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		semaphore <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-semaphore }()

			// Create a base request to the balancer
			req, err := http.NewRequest("GET", balancerURL, nil)
			if err != nil {
				return
			}

			// CRITICAL FIX: Inject the virtual host header so the balancer can route it
			req.Host = targetHost

			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				mu.Lock()
				results["FAILED_REQUESTS_OR_TIMEOUTS"]++
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return
			}

			serverID := strings.TrimSpace(string(body))

			mu.Lock()
			results[serverID]++
			mu.Unlock()
		}()
	}

	wg.Wait()
	duration := time.Since(startTime)

	// Final Report Output
	fmt.Println("\n============== FLOOD REPORT ==============")
	fmt.Printf("Target Domain:      %s\n", targetHost)
	fmt.Printf("Total Time Elapsed: %v\n", duration)
	fmt.Printf("Requests/Second:    %.2f\n\n", float64(totalRequests)/duration.Seconds())
	fmt.Println("Traffic Distribution Across Cluster:")

	for server, count := range results {
		fmt.Printf(" %s -> %d requests\n", server, count)
	}
	fmt.Println("==========================================")
}
