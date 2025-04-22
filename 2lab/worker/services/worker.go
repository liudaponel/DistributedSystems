package services

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"math"
	"strings"
	"time"

	"worker/models"
	"worker/publisher"

	"github.com/streadway/amqp"
)

const (
	defaultCharset    = "abcdefghijklmnopqrstuvwxyz0123456789"
	publishRetryDelay = 5 * time.Second
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

	for {
		// Проверяем главный контекст перед каждой попыткой подключения/потребления
		select {
		case <-ctx.Done():
			log.Println("Main context cancelled before attempting connection cycle.")
			return ctx.Err()
		default:
			// Продолжаем попытку
		}

		if !w.Publisher.IsConnected() {
			log.Println("Not connected. Attempting reconnect before consuming")
			// содержит цикл с переподключением
			err := w.Publisher.Reconnect(ctx)
			if err != nil {
				log.Printf("Reconnect failed: %v. Worker stopping.", err)
				return err
			}
			log.Println("Reconnect successful.")
		}

		log.Println("Attempting to establish consumer...")
		msgs, err := w.Publisher.ConsumeResults()
		if err != nil {
			log.Printf("Failed to establish consumer: %v. Will retry connection after delay...", err)
			// Ждем перед следующей попыткой во внешнем цикле
			select {
			case <-time.After(publishRetryDelay):
				continue // Начать следующую итерацию внешнего цикла for
			case <-ctx.Done():
				log.Println("Main context cancelled while waiting to retry consumer setup.")
				return ctx.Err()
			}
		}

		// внутренний цикл потребления сообщений
		log.Println("Consumer established successfully. Waiting for messages...")
	consumeLoop:
		for {
			select {
			case <-ctx.Done():
				log.Println("Main context cancelled while consuming. Worker stopping.")
				// Сообщение (если было взято) не Ack'ается, вернется в очередь
				return ctx.Err()
			case msg, ok := <-msgs:
				if !ok {
					log.Println("RabbitMQ message channel closed. Breaking consume loop to attempt reconnect.")
					break consumeLoop // Выходим из внутреннего цикла for, чтобы внешний цикл попробовал переподключиться
				}
				w.processNewTask(ctx, msg)
			}
		}

		// Если мы здесь, значит break consumeLoop был вызван (канал msgs закрылся)
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			log.Println("Main context cancelled immediately after channel closure.")
			return ctx.Err()
		}

	}
}

func (w *Worker) processNewTask(ctx context.Context, msg amqp.Delivery) {
	if len(msg.Body) == 0 {
		_ = msg.Nack(false, false)
		return
	}
	var task models.TaskRequest
	err := json.Unmarshal(msg.Body, &task)
	if err != nil {
		log.Printf("Failed to unmarshal message body: %v. Body: %s", err, string(msg.Body))
		_ = msg.Nack(false, false)
		return
	}
	if task.PartCount <= 0 {
		log.Printf("Received task with invalid PartCount <= 0. RequestId: %s, Part: %d/%d", task.RequestId, task.PartNumber, task.PartCount)
		_ = msg.Nack(false, false)
		return
	}

	log.Printf("Processing task for RequestId: %s, Part: %d/%d", task.RequestId, task.PartNumber, task.PartCount)
	foundWords := w.BruteForce(task)
	response := models.TaskResponse{RequestId: task.RequestId, Words: foundWords, PartNumber: task.PartNumber}
	body, _ := json.Marshal(response)

	log.Printf("Attempting to publish result for RequestId: %s, Part: %d...", task.RequestId, task.PartNumber)
	for { // Бесконечный цикл попыток отправки
		select {
		case <-ctx.Done():
			log.Printf("Context cancelled during publish attempt for task %s part %d. Aborting publish and exiting without Ack.", task.RequestId, task.PartNumber)
			return
		default:
			// Продолжаем попытку отправки
		}

		err = w.Publisher.PublishResponse(body)
		if err == nil {
			log.Printf("Successfully published result for RequestId: %s, Part: %d, words: %v", task.RequestId, task.PartNumber, response.Words)
			msg.Ack(false)
			return
		}

		log.Printf("Failed to publish result for task %s part %d: %v. Will retry after reconnect attempt.", task.RequestId, task.PartNumber, err)

		// Пытаемся переподключиться.
		log.Println("Attempting reconnect...")
		reconnectErr := w.Publisher.Reconnect(ctx)
		if reconnectErr != nil {
			log.Printf("Reconnect failed for task %s part %d because context was cancelled: %v. Exiting without Ack.", task.RequestId, task.PartNumber, reconnectErr)
			return
		}
		log.Println("Reconnect successful. Retrying publish")
		// Цикл for продолжится и снова попытается вызвать PublishResponse
		time.Sleep(1 * time.Second)

	}
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
