package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/streadway/amqp"
)

var ctx = context.Background()

func main() {
	r := gin.Default()

	// Redis conn
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// RabbitMQ conn
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("orders", false, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
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

	r.Run(":5002")
}
