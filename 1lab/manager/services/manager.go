package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"manager/models"
	"net/http"
	"sync"
	"time"
)

type Manager struct {
	Requests        map[string]*models.RequestInfo
	Mutex           sync.Mutex
	WorkerAddresses []string
}

func NewManager(workerAddresses []string) *Manager {
	return &Manager{
		Requests:        make(map[string]*models.RequestInfo),
		WorkerAddresses: workerAddresses,
	}
}

func (manager *Manager) CountCurrentConnections() int {
	count := 0
	for _, req := range manager.Requests {
		if req.Status == models.StatusInProgress {
			count++
		}
	}
	return count
}

func (manager *Manager) DistributeTasks(requestId string, req models.StartRequest) {
	manager.Mutex.Lock()
	manager.Requests[requestId].Status = models.StatusInProgress
	manager.Mutex.Unlock()

	countWorkers := len(manager.WorkerAddresses)
	manager.checkAvailableWorkers(countWorkers, requestId)

	wg := sync.WaitGroup{}
	for i, workerAddress := range manager.WorkerAddresses {
		wg.Add(1)
		workerReq := models.WorkerRequest{
			RequestId:  requestId,
			PartNumber: i,
			PartCount:  countWorkers,
			Hash:       req.Hash,
			MaxLength:  req.MaxLength,
		}
		go func(url string, req models.WorkerRequest) {
			defer wg.Done()
			manager.SendTask(url, req)
		}(workerAddress, workerReq)
	}
	wg.Wait()

	time.AfterFunc(10*time.Minute, func() {
		log.Printf("timeout Request %s\n", requestId)
		manager.Mutex.Lock()
		var status = manager.Requests[requestId]
		if status.Status == models.StatusInProgress && status.Received > 0 {
			manager.Requests[requestId].Status = models.StatusPartialReady
		} else {
			manager.Requests[requestId].Status = models.StatusError
		}
		manager.Mutex.Unlock()
	})
}

func (manager *Manager) checkAvailableWorkers(countWorkers int, requestId string) {
	if countWorkers == 0 {
		log.Printf("No available workers for request %s\n", requestId)
		manager.Mutex.Lock()
		manager.Requests[requestId].Status = models.StatusError
		manager.Mutex.Unlock()
		return
	}
}

func (manager *Manager) SendTask(workerAddress string, req models.WorkerRequest) bool {
	data, _ := json.Marshal(req)
	_, err := http.Post(workerAddress+"/internal/api/worker/hash/crack/task", "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Failed to send task to %s: %v\n", workerAddress, err)
		return false
	}
	log.Printf("Sent task %s to %s\n", req.RequestId, workerAddress)
	return true
}

func (manager *Manager) CheckWorkerResponse(res models.WorkerResponse) {
	manager.Mutex.Lock()
	defer manager.Mutex.Unlock()

	if requestInfo, exists := manager.Requests[res.RequestId]; exists {
		requestInfo.Data = append(requestInfo.Data, res.Words...)
		requestInfo.Received++
		log.Printf("Received %d from %d workers, request %s\n", requestInfo.Received, requestInfo.Total, res.RequestId)
		if requestInfo.Received == requestInfo.Total {
			requestInfo.Status = models.StatusReady
			log.Printf("Request %s completed\n", res.RequestId)
		} else {
			requestInfo.Status = models.StatusPartialReady
		}
	}
}

func (manager *Manager) GetRequestStatus(requestId string) (models.StatusResponse, bool) {
	manager.Mutex.Lock()
	defer manager.Mutex.Unlock()

	if requestInfo, exists := manager.Requests[requestId]; exists {
		var data *[]string = nil
		if len(requestInfo.Data) > 0 {
			data = &requestInfo.Data
		}
		res := models.StatusResponse{
			Status:   requestInfo.Status,
			Progress: manager.getTotalProgress(requestId),
			Data:     data,
		}
		if res.Status == models.StatusReady {
			res.Progress = 100
		}
		return res, true
	}
	return models.StatusResponse{}, false
}

func (manager *Manager) getTotalProgress(requestId string) int {
	var totalProgress float64

	for _, workerAddress := range manager.WorkerAddresses {
		progress, err := manager.SendProgressRequest(workerAddress, requestId)
		if err != nil {
			log.Printf("Error getting progress from worker %s: %v", workerAddress, err)
			continue
		}
		totalProgress += progress
	}

	// Вычисляем средний прогресс
	if len(manager.WorkerAddresses) > 0 {
		return int(totalProgress / float64(len(manager.WorkerAddresses)))
	}
	return 0
}

func (manager *Manager) SendProgressRequest(workerAddress string, requestId string) (float64, error) {
	resp, err := http.Get(fmt.Sprintf("%s/internal/api/worker/hash/crack/progress?requestId=%s", workerAddress, requestId))
	if err != nil {
		return 0, fmt.Errorf("failed to send request to worker %s: %v", workerAddress, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("worker %s returned status code %d", workerAddress, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body from worker %s: %v", workerAddress, err)
	}

	var progressResponse models.ProgressResponse
	if err := json.Unmarshal(body, &progressResponse); err != nil {
		return 0, fmt.Errorf("failed to parse JSON from worker %s: %v", workerAddress, err)
	}

	return progressResponse.Progress, nil
}
