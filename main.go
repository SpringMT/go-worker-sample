package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Data struct {
	value string
}

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
	workerDone := make(chan struct{})
	go func(stopCh <-chan struct{}, doneCh chan<- struct{}) {
		defer close(doneCh)
		for {
			select {
			case <-stopCh:
				log.Println("Stopping worker...")
				time.Sleep(3 * time.Second)
				log.Println("Worker stopped.")
				return
			default:
				d, err := rb.Dequeue()
				if err != nil {
					log.Printf("Error: %v", err)
				} else {
					log.Printf("Dequeued: %v", d)
				}
				time.Sleep(2 * time.Second)
			}
		}
	}(stopWorker, workerDone)
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
		<-workerDone
	}
}
