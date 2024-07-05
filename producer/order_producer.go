package producer

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
)

type orderProducer struct{}

func NewOrderProducer() *orderProducer {
	return &orderProducer{}
}

func (o *orderProducer) OrderProducer(queue, msg string) {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("failed to loading .env file: %v", err)
	}

	// เชื่อมต่อกับ RabbitMQ server
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// เปิด channel สำหรับสื่อสารกับ RabbitMQ
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open a channel: %v", err)
	}
	defer ch.Close()

	// ประกาศ queue ที่จะส่งข้อมูลไป
	q, err := ch.QueueDeclare(
		queue, // ชื่อของ queue
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("failed to declare a queue: %v", err)
	}

	// ส่งข้อความไปยัง queue
	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		})
	if err != nil {
		log.Fatalf("failed to publish a message: %v", err)
	}

	fmt.Println("Successfully published message to RabbitMQ")
}
