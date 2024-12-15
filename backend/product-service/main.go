package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient *mongo.Client
	redisClient *redis.Client
	ctx         = context.Background()
)

func init() {
	var err error

	// MongoDB conn
	mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://my_user:my_password@mongodb:27017"))
	if err != nil {
		panic(err)
	}

	// Redis conn
	redisClient = redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})

	//create test product
	initializeTestProduct()
}

func main() {
	r := gin.Default()

	r.GET("/products", GetProducts)
	r.POST("/products", AddProduct)

	r.Run(":5001")
}

type Product struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
}

// create test product and save MongoDB and Redis
func initializeTestProduct() {
	// Test product
	testProduct := Product{
		ID:       "1",
		Name:     "Test Product",
		Price:    99.99,
		Category: "Test Category",
	}

	// save test product to MongoDB
	collection := mongoClient.Database("product_db").Collection("products")
	_, err := collection.InsertOne(ctx, testProduct)
	if err != nil {
		log.Printf("Error occurred while saving test product to MongoDB: %v\n", err)
	} else {
		log.Println("test product saved to MongoDB.")
	}

	// save test product to Redis
	cacheKey := "products"
	var products []Product
	products = append(products, testProduct)

	productsJSON, _ := json.Marshal(products)
	err = redisClient.Set(ctx, cacheKey, productsJSON, 10*time.Minute).Err()
	if err != nil {
		log.Printf("occurred while saving test product to Redis: %v\n", err)
	} else {
		log.Println("test product saved to Redis.")
	}
}

// list products
func GetProducts(c *gin.Context) {
	cacheKey := "products"
	productsCache, err := redisClient.Get(ctx, cacheKey).Result()
	if err != redis.Nil {
		log.Printf("products cached from Redis")
	}
	if err == redis.Nil {
		// if not in Redis get from MongoDB
		collection := mongoClient.Database("product_db").Collection("products")
		cursor, err := collection.Find(ctx, bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var products []Product
		for cursor.Next(ctx) {
			var product Product
			cursor.Decode(&product)
			products = append(products, product)
		}

		// save to Redis
		productsJSON, _ := json.Marshal(products)
		redisClient.Set(ctx, cacheKey, productsJSON, 10*time.Minute)
		c.JSON(http.StatusOK, products)
		return
	}

	// get from Redis
	var products []Product
	json.Unmarshal([]byte(productsCache), &products)
	c.JSON(http.StatusOK, products)
}

// create product
func AddProduct(c *gin.Context) {
	var product Product
	if err := c.BindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	collection := mongoClient.Database("product_db").Collection("products")
	_, err := collection.InsertOne(ctx, product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// delete cache
	redisClient.Del(ctx, "products")
	c.JSON(http.StatusCreated, gin.H{"message": "Product added"})
}
