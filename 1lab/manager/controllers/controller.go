package controllers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"

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
	http.HandleFunc("/internal/api/manager/hash/crack/request", controller.HandleWorkerResponse)
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

	requestId := uuid.New().String()
	// Создаем запись о запросе со статусом NEW
	controller.Manager.Mutex.Lock()
	controller.Manager.Requests[requestId] = &models.RequestInfo{
		Status: models.StatusNew,
		Total:  len(controller.Manager.WorkerAddresses),
	}
	controller.Manager.Mutex.Unlock()

	log.Printf("New request: %s \n", requestId)
	//распределяем задачи по воркерам и переводим в статус IN_PROGRESS
	go controller.Manager.DistributeTasks(requestId, req)
	json.NewEncoder(w).Encode(models.StartResponse{RequestId: requestId})
}

func (controller *Controller) HandleWorkerResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var res models.WorkerResponse
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		http.Error(w, "Invalid response", http.StatusBadRequest)
		return
	}

	controller.Manager.CheckWorkerResponse(res)
	w.WriteHeader(http.StatusOK)
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
