package main

import (
	"log"
	"net/http"
	"os"

	"worker/controllers"
	"worker/services"
)

func main() {
	managerAddress := os.Getenv("MANAGER_ADDRESS")
	port := os.Getenv("PORT")

	worker := services.NewWorker(managerAddress)
	controller := controllers.NewController(worker)
	controller.RegisterRoutes()

	log.Printf("Worker is running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
