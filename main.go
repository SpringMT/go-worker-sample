package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/semaphore"
)

type Data struct {
	value string
}

const maxWorkers = 5

func main() {
	rb := NewRingBuffer[Data](1000)
	count := 0
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		count++
		rb.Enqueue(Data{fmt.Sprintf("Hello, world! %d", count)})
		fmt.Fprintf(w, "Welcome to my website!")
	})

	serverAddr := ":8000"
	srv := &http.Server{Addr: serverAddr}

	done := make(chan error)
	go func() {
		log.Printf("Starting HTTP server on %s", serverAddr)
		done <- srv.ListenAndServe()
	}()

	stopWorker := make(chan struct{})
	sem := semaphore.NewWeighted(maxWorkers)
	var wg sync.WaitGroup

	for i := 0; i < maxWorkers; i++ {
		if err := sem.Acquire(context.Background(), 1); err != nil {
			log.Printf("Failed to acquire semaphore: %v", err)
			break
		}
		wg.Add(1)
		go func(stopCh <-chan struct{}) {
			defer func() {
				wg.Done()
				sem.Release(1)
			}()
			for {
				select {
				case <-stopCh:
					log.Println("Stopping worker...")
					time.Sleep(3 * time.Second)
					log.Println("Worker stopped.")
					return
				default:
					fmt.Printf("Currently %d goroutines are running.\n", runtime.NumGoroutine())
					d, err := rb.Dequeue()
					if err != nil {
						log.Printf("Error: %v", err)
					} else {
						log.Printf("Dequeued: %v", d)
					}
					time.Sleep(2 * time.Second)
				}
			}
		}(stopWorker)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	select {
	case err := <-done:
		slog.ErrorContext(ctx, "failed to run the process", err)
		os.Exit(1)
	case <-ctx.Done():
		slog.InfoContext(ctx, "Signalがきた")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		slog.InfoContext(ctx, "shutdownを開始")
		defer cancel()
		srv.Shutdown(ctx)
		slog.InfoContext(ctx, "worker shutdownを開始")
		close(stopWorker)
		wg.Wait()
	}
}
