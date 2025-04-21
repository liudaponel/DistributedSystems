package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"manager/controllers"
	"manager/services"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	rabbitURL := os.Getenv("RABBITMQ_URL")

	client := initMongo()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Disconnect(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	manager, err := services.NewManager(client, rabbitURL)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}
	controller := controllers.NewController(manager)
	controller.RegisterRoutes()

	log.Println("Manager started on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initMongo() *mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://mongo-primary:27017,mongo-secondary1:27017,mongo-secondary2:27017/?replicaSet=rs0"))
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB successfully")
	return client
}
