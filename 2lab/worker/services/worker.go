package services

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"math"
	"strings"

	"worker/models"
	"worker/publisher"

	"github.com/streadway/amqp"
)

const (
	defaultCharset = "abcdefghijklmnopqrstuvwxyz0123456789"
)

type Worker struct {
	Charset   string
	Publisher publisher.Publisher
}

func NewWorker(rabbitURL string) (*Worker, error) {
	newPublisher, err := publisher.NewPublisher(rabbitURL)
	if err != nil {
		return nil, err
	}

	w := &Worker{
		Charset:   defaultCharset,
		Publisher: *newPublisher,
	}

	log.Println("Worker initialized and connected to RabbitMQ.")
	return w, nil
}

func (w *Worker) StartConsuming(ctx context.Context) error {
	msgs, err := w.Publisher.ConsumeResults()
	if err != nil {
		return err
	}
	log.Println("Consumer started. Waiting for messages or context cancellation...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Worker stopping")
			return ctx.Err()
		case msg := <-msgs:
			w.processNewTask(msg)
		}
	}
}

func (w *Worker) processNewTask(msg amqp.Delivery) {
	var task models.TaskRequest
	json.Unmarshal(msg.Body, &task)
	log.Printf("Processing task for RequestId: %s, Part: %d/%d", task.RequestId, task.PartNumber, task.PartCount)

	foundWords := w.BruteForce(task)

	response := models.TaskResponse{
		RequestId:  task.RequestId,
		Words:      foundWords,
		PartNumber: task.PartNumber,
	}
	body, _ := json.Marshal(response)

	err := w.Publisher.PublishResponse(body)
	if err != nil {
		log.Printf("Failed to publish result for task %s part %d: %v", task.RequestId, task.PartNumber, err)
		return
	}
	log.Printf("Sent result for RequestId: %s, Part: %d, words: %v", task.RequestId, task.PartNumber, response.Words)

	msg.Ack(false)
}

func (w *Worker) BruteForce(task models.TaskRequest) []string {
	targetHash := strings.ToLower(task.Hash)
	foundWords := []string{}

	var totalCombinations int64 = 0
	for length := 1; length <= task.MaxLength; length++ {
		totalCombinations += int64(math.Pow(float64(len(w.Charset)), float64(length)))
	}

	partSize := totalCombinations / int64(task.PartCount)
	indexStart := int64(task.PartNumber) * partSize
	indexEnd := indexStart + partSize
	var indexForAllCombinations int64 = 0
	for length := 1; length <= task.MaxLength; length++ {
		countCombinationsForLength := int64(math.Pow(float64(len(w.Charset)), float64(length)))

		if indexForAllCombinations+countCombinationsForLength <= indexStart {
			indexForAllCombinations += countCombinationsForLength
			continue
		}

		if indexForAllCombinations >= indexEnd {
			break
		}

		word := make([]byte, length)
		w.CheckCombinations(word, 0, targetHash, &foundWords, indexStart, indexEnd, &indexForAllCombinations)
	}

	return foundWords
}

func (w *Worker) CheckCombinations(word []byte, position int, targetHash string,
	words *[]string, indexStart, indexEnd int64, indexForAllCombinations *int64) {

	if position == len(word) {
		if *indexForAllCombinations >= indexStart && *indexForAllCombinations < indexEnd {
			hash := md5.Sum(word)
			if hex.EncodeToString(hash[:]) == targetHash {
				*words = append(*words, string(word))
			}
		}

		(*indexForAllCombinations)++
		return
	}

	if *indexForAllCombinations+int64(math.Pow(float64(len(w.Charset)),
		float64(len(word)-position))) <= indexStart {

		*indexForAllCombinations += int64(math.Pow(float64(len(w.Charset)),
			float64(len(word)-position)))
		return
	}

	if *indexForAllCombinations >= indexEnd {

		*indexForAllCombinations += int64(math.Pow(float64(len(w.Charset)),
			float64(len(word)-position)))
		return
	}

	for i := 0; i < len(w.Charset); i++ {
		word[position] = w.Charset[i]
		w.CheckCombinations(word, position+1, targetHash, words, indexStart, indexEnd, indexForAllCombinations)
	}
}
