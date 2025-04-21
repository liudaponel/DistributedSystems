package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"worker/services"
)

func main() {
	rabbitURL := os.Getenv("RABBITMQ_URL")

	worker, err := services.NewWorker(rabbitURL)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	err = worker.StartConsuming(ctx)
	if err != nil {
		log.Printf("Failed to start consuming: %v", err)
	}
}
