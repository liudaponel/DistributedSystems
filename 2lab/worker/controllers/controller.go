package controllers

import (
	"encoding/json"
	"log"
	"net/http"

	"worker/models"
	"worker/services"
)

type Controller struct {
	Worker *services.Worker
}

func NewController(worker *services.Worker) *Controller {
	return &Controller{
		Worker: worker,
	}
}

func (controller *Controller) RegisterRoutes() {
	http.HandleFunc("/internal/api/worker/hash/crack/task", controller.GetTask)
	http.HandleFunc("/internal/api/worker/hash/crack/progress", controller.GetProgress)
}

func (controller *Controller) GetTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var task models.TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		log.Printf("Failed to decode request: %v", err)
		return
	}
	log.Printf("Received task with request id %s, part %d of %d", task.RequestId, task.PartNumber, task.PartCount)
	go controller.Worker.Work(task)
	w.WriteHeader(http.StatusAccepted)
}

func (controller *Controller) GetProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	requestId := r.URL.Query().Get("requestId")
	progress, exists := controller.Worker.GetProgress(requestId)
	if exists {
		json.NewEncoder(w).Encode(progress)
	} else {
		http.Error(w, "Request not found", http.StatusNotFound)
	}
}
