package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/streadway/amqp"
)

var ctx = context.Background()

func waitForService(serviceName string, maxAttempts int, sleep time.Duration) {
	for i := 0; i < maxAttempts; i++ {
		if i > 0 {
			time.Sleep(sleep)
		}

		var err error
		switch serviceName {
		case "redis":
			rdb := redis.NewClient(&redis.Options{
				Addr: "redis:6379",
			})
			_, err = rdb.Ping(ctx).Result()
		case "rabbitmq":
			url := "amqp://guest:guest@rabbitmq:5672/"
			fmt.Printf("Attempting to connect to RabbitMQ at: %s\n", url)
			_, err = amqp.Dial(url)
		}

		if err == nil {
			fmt.Printf("%s is ready\n", serviceName)
			return
		}

		fmt.Printf("Error connecting to %s (attempt %d/%d): %v\n", serviceName, i+1, maxAttempts, err)
	}
	log.Fatalf("Could not connect to %s after %d attempts", serviceName, maxAttempts)
}

func main() {
	// Increase initial wait time to ensure services are up
	time.Sleep(10 * time.Second)

	fmt.Println("Starting order service...")

	// Wait for services to be ready first
	waitForService("redis", 10, 2*time.Second)
	waitForService("rabbitmq", 10, 2*time.Second)

	fmt.Println("Services are ready, initializing application...")

	r := gin.Default()

	// Redis conn
	rdb := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})

	// RabbitMQ conn
	rabbitmqURL := "amqp://guest:guest@rabbitmq:5672/"
	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("orders", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	// create order
	r.POST("/orders", func(c *gin.Context) {
		var order map[string]string
		if err := c.BindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		id := order["id"]
		rdb.HSet(ctx, "orders", id, order["product"])
		ch.Publish("", q.Name, false, false, amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("New order created"),
		})
		c.JSON(http.StatusCreated, gin.H{"message": "Order created"})
	})

	fmt.Println("Order service is ready to accept requests on port 5002")
	r.Run(":5002")
}
