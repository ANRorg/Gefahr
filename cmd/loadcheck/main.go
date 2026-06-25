// Command loadcheck runs a repeatable concurrency and leak smoke check.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	target := flag.String("target", "http://localhost:8080/", "proxy URL")
	metricsURL := flag.String("metrics", "http://127.0.0.1:9090/metrics", "admin metrics URL")
	requests := flag.Int("requests", 1000, "total requests")
	concurrency := flag.Int("concurrency", 20, "parallel workers")
	maxGoroutineGrowth := flag.Int64("max-goroutine-growth", 20, "allowed settled goroutine increase")
	flag.Parse()
	if *requests <= 0 || *concurrency <= 0 || *maxGoroutineGrowth < 0 {
		fatal(fmt.Errorf("requests and concurrency must be positive and max-goroutine-growth must be non-negative"))
	}
	client := &http.Client{Timeout: 10 * time.Second}
	warmupRequests := min(*requests, max(1000, *concurrency*100))
	warmupFailures, _ := runLoad(client, *target, warmupRequests, *concurrency)
	if warmupFailures > 0 {
		fatal(fmt.Errorf("load check warm-up observed %d failures", warmupFailures))
	}
	client.CloseIdleConnections()
	time.Sleep(time.Second)
	before, err := scrape(client, *metricsURL)
	if err != nil {
		fatal(err)
	}
	failures, elapsed := runLoad(client, *target, *requests, *concurrency)
	client.CloseIdleConnections()
	time.Sleep(time.Second)
	after, err := scrape(client, *metricsURL)
	if err != nil {
		fatal(err)
	}
	growth := int64(after["go_goroutines"] - before["go_goroutines"])
	fmt.Printf("requests=%d failures=%d duration=%s rate=%.1f/s goroutines_before=%.0f goroutines_after=%.0f heap_before=%.0f heap_after=%.0f\n", *requests, failures, elapsed.Round(time.Millisecond), float64(*requests)/elapsed.Seconds(), before["go_goroutines"], after["go_goroutines"], before["go_memstats_alloc_bytes"], after["go_memstats_alloc_bytes"])
	if failures > 0 {
		fatal(fmt.Errorf("load check observed %d failures", failures))
	}
	if growth > *maxGoroutineGrowth {
		fatal(fmt.Errorf("settled goroutine growth %d exceeds %d", growth, *maxGoroutineGrowth))
	}
}

func runLoad(client *http.Client, target string, requests, concurrency int) (int64, time.Duration) {
	var next, failures atomic.Int64
	started := time.Now()
	var workers sync.WaitGroup
	for range concurrency {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				index := next.Add(1)
				if index > int64(requests) {
					return
				}
				request, err := http.NewRequest(http.MethodGet, target, nil)
				if err != nil {
					failures.Add(1)
					continue
				}
				request.Header.Set("Cache-Control", "no-cache")
				resp, err := client.Do(request)
				if err != nil {
					failures.Add(1)
					continue
				}
				_, copyErr := io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if copyErr != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
					failures.Add(1)
				}
			}
		}()
	}
	workers.Wait()
	return failures.Load(), time.Since(started)
}

func scrape(client *http.Client, target string) (map[string]float64, error) {
	request, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	if token := os.Getenv("GOPROXY_ADMIN_TOKEN"); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metrics status %d", resp.StatusCode)
	}
	values := map[string]float64{}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.Contains(line, "{") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		value, err := strconv.ParseFloat(fields[1], 64)
		if err == nil {
			values[fields[0]] = value
		}
	}
	return values, scanner.Err()
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
