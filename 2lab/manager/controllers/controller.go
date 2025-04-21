package controllers

import (
	"encoding/json"
	"log"
	"net/http"

	"manager/models"
	"manager/services"
)

const maxConnections = 2

type Controller struct {
	Manager *services.Manager
}

func NewController(manager *services.Manager) *Controller {
	return &Controller{
		Manager: manager,
	}
}

func (controller *Controller) RegisterRoutes() {
	http.HandleFunc("/api/hash/crack", controller.HandleStartRequest)
	http.HandleFunc("/api/hash/status", controller.GetStatus)
}

func (controller *Controller) HandleStartRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req models.StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if controller.Manager.CountCurrentConnections() >= maxConnections {
		http.Error(w, "Too many requests", http.StatusTooManyRequests)
		return
	}

	requestInfo := controller.Manager.Repository.SaveNewRequest(req, 3)
	log.Printf("New request: %s \n", requestInfo.ID)

	go controller.Manager.StartCrackRequest(req, requestInfo.ID)
	json.NewEncoder(w).Encode(models.StartResponse{RequestId: requestInfo.ID})
}

func (controller *Controller) GetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	requestId := r.URL.Query().Get("requestId")
	status, exists := controller.Manager.GetRequestStatus(requestId)

	if exists {
		json.NewEncoder(w).Encode(status)
	} else {
		http.Error(w, "Request not found", http.StatusNotFound)
	}
}
