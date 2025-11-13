package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/beheryahmed1991/subscription-service.git/internal/db"
	"github.com/beheryahmed1991/subscription-service.git/internal/subscription"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	_ "github.com/beheryahmed1991/subscription-service.git/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Subscription Service
// @version 1.0
// @description REST API for managing user subscriptions
// @host localhost:8080
func main() {

	_ = godotenv.Load("../.env", ".env")
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		log.Fatal("postgres_url is not set")
	}

	ctx := context.Background()
	database, err := db.New(ctx, db.Config{
		URL:             dsn,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	defer database.Close()

	router := gin.Default()

	router.GET("/hello", func(c *gin.Context) {
		c.String(200, "Hello, ahmed!")
	})

	subRepo := subscription.NewRepository(database)
	subHandler := subscription.NewHandler(subRepo)
	subHandler.RegisterRoutes(router)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	if err := router.Run(":8080"); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
