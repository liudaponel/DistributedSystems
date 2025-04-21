package publisher

import (
	"fmt"

	"github.com/streadway/amqp"
)

const (
	rabbitMQDefaultURL = "amqp://guest:guest@rabbitmq:5672/"
	taskExchangeName   = "tasks_exchange"     // Имя exchange (direct)
	taskQueueName      = "tasks_queue"        // Очередь задач для воркера
	resultQueueName    = "results_queue"      // Очередь результатов (для объявления, но не для потребления)
	taskRoutingKey     = "task.crack.request" // Ключ для получения задач
	resultRoutingKey   = "task.crack.result"  // Ключ для отправки результатов
)

type Publisher struct {
	rabbitConn *amqp.Connection
	rabbitChan *amqp.Channel
}

func NewPublisher(rabbitURL string) (*Publisher, error) {
	if rabbitURL == "" {
		rabbitURL = rabbitMQDefaultURL
	}

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	err = ch.ExchangeDeclare(
		taskExchangeName, // name
		"direct",         // type
		true,             // durable
		false,            // auto-deleted
		false,            // internal
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange '%s': %w", taskExchangeName, err)
	}

	ch.QueueDeclare(
		taskQueueName, // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	ch.QueueBind(
		taskQueueName,    // queue name
		taskRoutingKey,   // routing key
		taskExchangeName, // exchange
		false,
		nil,
	)

	ch.QueueDeclare(
		resultQueueName, // name
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		nil,             // arguments
	)
	ch.QueueBind(
		resultQueueName,  // queue name
		resultRoutingKey, // routing key for results
		taskExchangeName, // exchange
		false,
		nil)

	return &Publisher{
		rabbitConn: conn,
		rabbitChan: ch,
	}, nil
}

func (p *Publisher) ConsumeResults() (<-chan amqp.Delivery, error) {
	if p.rabbitChan == nil {
		return nil, fmt.Errorf("cannot consume results, RabbitMQ channel is not open or nil")
	}

	msgs, _ := p.rabbitChan.Consume(
		taskQueueName,      // queue
		"manager-consumer", // consumer tag
		false,              // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	return msgs, nil
}

func (p *Publisher) PublishResponse(body []byte) error {
	return p.rabbitChan.Publish(
		taskExchangeName, // exchange
		resultRoutingKey, // routing key for results
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		})
}
