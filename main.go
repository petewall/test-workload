package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/armon/go-metrics"
	"github.com/armon/go-metrics/prometheus"
	"github.com/fxtlabs/primes"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func metricsAndLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("IN  [%s] %s %s %s\n", r.Method, r.Host, r.URL.Path, r.URL.RawQuery)
		start := time.Now()
		record := &StatusSavingResponseWriter{ResponseWriter: w, StatusCode: http.StatusOK}

		next.ServeHTTP(record, r)

		labels := []metrics.Label{
			{Name: "method", Value: r.Method},
			{Name: "route", Value: r.RequestURI},
			{Name: "code", Value: fmt.Sprintf("%d", record.StatusCode)},
		}

		duration := time.Since(start)
		log.Printf("OUT [%s] %s %s %s: %d %dms\n", r.Method, r.Host, r.URL.Path, r.URL.RawQuery, record.StatusCode, duration.Milliseconds())
		metrics.AddSampleWithLabels([]string{"request_duration"}, float32(duration.Milliseconds()), labels)
	})
}

type StatusSavingResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (r *StatusSavingResponseWriter) WriteHeader(status int) {
	r.StatusCode = status
	r.ResponseWriter.WriteHeader(status)
}

type Response struct {
	Message  string `json:"message"`
	Duration int    `json:"duration"`
}

func ShouldFail() bool {
	return rand.Intn(100) < config.ErrorRate
}

func NumPrimes() int {
	primeRange := config.CPU.MaxPrimesCalclated - config.CPU.MinPrimesCalclated
	return config.CPU.MinPrimesCalclated + rand.Intn(primeRange)
}

func RequestHandler(w http.ResponseWriter, r *http.Request) {
	count := NumPrimes()
	log.Printf("Calculating %d primes...\n", count)
	primes.Sieve(count)

	if ShouldFail() {
		fmt.Println("Oops! Erroring...")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := &Response{
		Message:  "Hello world!",
		Duration: count,
	}

	data, _ := json.Marshal(response)
	_, _ = w.Write(data)
}

func main() {
	watcher, err := LoadConfig(os.Getenv("CONFIG_FILE_PATH"))
	if err != nil {
		log.Fatal("failed to load configuration:", err)
	}
	defer watcher.Close()
	mon, _ := prometheus.NewPrometheusSink()
	_, _ = metrics.NewGlobal(metrics.DefaultConfig("test-workload"), mon)

	mux := http.NewServeMux()
	mux.HandleFunc("/", RequestHandler)
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    ":3000",
		Handler: metricsAndLoggingMiddleware(mux),
	}
	log.Println("Starting server at", server.Addr)
	log.Fatal(server.ListenAndServe())
}
