package main

import (
	"log"
	"net/http"
	"os"

	"manager/controllers"
	"manager/services"
)

func main() {
	var workerAddresses = []string{
		os.Getenv("WORKER1_ADDRESS"),
		os.Getenv("WORKER2_ADDRESS"),
		os.Getenv("WORKER3_ADDRESS"),
	}

	manager := services.NewManager(workerAddresses)
	controller := controllers.NewController(manager)

	controller.RegisterRoutes()

	log.Println("Manager started on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
