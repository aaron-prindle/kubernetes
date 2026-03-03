package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"
)

func main() {
	target := flag.String("target", "", "API Server URL")
	token := flag.String("token", "", "ServiceAccount Token")
	concurrency := flag.Int("concurrency", 100, "Number of parallel goroutines")
	duration := flag.Duration("duration", 30*time.Second, "How long to run the test")
	flag.Parse()

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 1000,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	// Latency tracking
	var mu sync.Mutex
	latencies := make([]time.Duration, 0, 100000)

	fmt.Printf("Blasting %d workers at %s for %v...\n", *concurrency, *target, *duration)

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			workerID := fmt.Sprintf("worker-%d", id)
			for {
				select {
				case <-ctx.Done():
					return
				default:
					randVal := rand.Intn(1000000)
					manager := fmt.Sprintf("brutal-%s-%d", workerID, randVal)
					payload := []byte(fmt.Sprintf(`{
						"apiVersion": "v1",
						"kind": "ConfigMap",
						"metadata": {"name": "brutal-cm"},
						"data": {"key-%d": "value-%d"}
					}`, randVal, randVal))

					url := fmt.Sprintf("%s/api/v1/namespaces/default/configmaps/brutal-cm?fieldManager=%s", *target, manager)
					req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(payload))
					req.Header.Set("Content-Type", "application/apply-patch+yaml")
					req.Header.Set("Authorization", "Bearer "+*token)

					start := time.Now()
					resp, err := client.Do(req)
					elapsed := time.Since(start)

					if err == nil {
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
						if resp.StatusCode == http.StatusOK {
							mu.Lock()
							latencies = append(latencies, elapsed)
							mu.Unlock()
						}
					}
				}
			}
		}(i)
	}

	wg.Wait()
	
	if len(latencies) == 0 {
		fmt.Println("No successful requests recorded.")
		return
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	
	var total time.Duration
	for _, l := range latencies {
		total += l
	}

	fmt.Println("\n==========================================================")
	fmt.Printf(" WRITE LATENCY RESULTS (%d requests)\n", len(latencies))
	fmt.Println("==========================================================")
	fmt.Printf("  Average: %v\n", total/time.Duration(len(latencies)))
	fmt.Printf("  P50:     %v\n", latencies[len(latencies)/2])
	fmt.Printf("  P95:     %v\n", latencies[len(latencies)*95/100])
	fmt.Printf("  P99:     %v\n", latencies[len(latencies)*99/100])
	fmt.Println("==========================================================\n")
}
