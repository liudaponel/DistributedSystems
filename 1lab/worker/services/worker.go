package services

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"

	"worker/models"
)

type Worker struct {
	ManagerAddress string
	Charset        string
	ProgressMap    map[string]*models.TaskProgress
	Mutex          sync.Mutex
}

func NewWorker(managerAddress string) *Worker {
	return &Worker{
		ManagerAddress: managerAddress,
		Charset:        "abcdefghijklmnopqrstuvwxyz0123456789",
		ProgressMap:    make(map[string]*models.TaskProgress),
	}
}

func (worker *Worker) Work(task models.TaskRequest) {
	// Инициализируем прогресс
	worker.Mutex.Lock()
	worker.ProgressMap[task.RequestId] = &models.TaskProgress{
		PartNumber: task.PartNumber,
		Progress:   0,
	}
	worker.Mutex.Unlock()

	words := worker.BruteForce(task)
	if len(words) != 0 {
		log.Printf("Found words RequestId: %s, part: %d, words: %v",
			task.RequestId, task.PartNumber, words)
	}
	worker.SendResponse(task.RequestId, task.PartNumber, words)
}

func (worker *Worker) GetProgress(requestId string) (models.ProgressResponse, bool) {
	if progress, exists := worker.ProgressMap[requestId]; exists {
		return models.ProgressResponse{
			RequestId:  requestId,
			PartNumber: progress.PartNumber,
			Progress:   progress.Progress,
		}, true
	}
	return models.ProgressResponse{}, false
}

func (worker *Worker) BruteForce(task models.TaskRequest) []string {
	hash := strings.ToLower(task.Hash)
	words := []string{}
	var totalCombinations int64 = 0
	for length := 1; length <= task.MaxLength; length++ {
		totalCombinations += int64(math.Pow(float64(len(worker.Charset)), float64(length)))
	}
	var progressRefresh = totalCombinations / 100

	partSize := totalCombinations / int64(task.PartCount)
	indexStart := int64(task.PartNumber) * partSize
	indexEnd := indexStart + partSize
	log.Printf("Start brute force from %d to %d", indexStart, indexEnd)

	var indexForAllCombinations int64 = 0
	for length := 1; length <= task.MaxLength; length++ {
		countCombinationsForLength := int64(math.Pow(float64(len(worker.Charset)), float64(length)))

		if indexForAllCombinations+countCombinationsForLength <= indexStart {
			indexForAllCombinations += countCombinationsForLength
			continue
		}

		if indexForAllCombinations >= indexEnd {
			break
		}

		word := make([]byte, length)
		worker.CheckCombinations(task.RequestId, progressRefresh, word, 0, hash, &words, indexForAllCombinations,
			indexStart, indexEnd, &indexForAllCombinations)
	}

	return words
}

func (worker *Worker) CheckCombinations(requestId string, progressRefresh int64, word []byte,
	position int, targetHash string, words *[]string, currentIndex, indexStart,
	indexEnd int64, indexForAllCombinations *int64) {

	if position == len(word) {
		if *indexForAllCombinations >= indexStart && *indexForAllCombinations < indexEnd {
			hash := md5.Sum(word)
			if hex.EncodeToString(hash[:]) == targetHash {
				*words = append(*words, string(word))
			}
		}

		(*indexForAllCombinations)++

		// Обновляем прогресс
		if *indexForAllCombinations%progressRefresh == 0 {
			worker.Mutex.Lock()
			if progress, ok := worker.ProgressMap[requestId]; ok {
				progress.Progress = float64(*indexForAllCombinations-indexStart) / float64(indexEnd-indexStart) * 100
			}
			worker.Mutex.Unlock()
		}

		return
	}

	if *indexForAllCombinations+int64(math.Pow(float64(len(worker.Charset)),
		float64(len(word)-position))) <= indexStart {

		*indexForAllCombinations += int64(math.Pow(float64(len(worker.Charset)),
			float64(len(word)-position)))
		return
	}

	if *indexForAllCombinations >= indexEnd {

		*indexForAllCombinations += int64(math.Pow(float64(len(worker.Charset)),
			float64(len(word)-position)))
		return
	}

	for i := 0; i < len(worker.Charset); i++ {
		word[position] = worker.Charset[i]
		worker.CheckCombinations(requestId, progressRefresh, word, position+1, targetHash, words,
			currentIndex, indexStart, indexEnd, indexForAllCombinations)
	}
}

func (worker *Worker) SendResponse(requestID string, partNumber int, words []string) {
	response := models.TaskResponse{
		RequestId:  requestID,
		Words:      words,
		PartNumber: partNumber,
	}

	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	req, err := http.NewRequest(
		http.MethodPatch,
		worker.ManagerAddress+"/internal/api/manager/hash/crack/request",
		bytes.NewBuffer(data),
	)
	if err != nil {
		log.Fatalf("Error creating PATCH request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending PATCH request: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("Sent response to manager for task %s, part %d, status: %s",
		requestID, partNumber, resp.Status)
}
