package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	docs "github.com/beheryahmed1991/subscription-service.git/docs"
	"github.com/beheryahmed1991/subscription-service.git/internal/config"
	"github.com/beheryahmed1991/subscription-service.git/internal/db"
	"github.com/beheryahmed1991/subscription-service.git/internal/logger"
	"github.com/beheryahmed1991/subscription-service.git/internal/middleware"
	"github.com/beheryahmed1991/subscription-service.git/internal/migrate"
	"github.com/beheryahmed1991/subscription-service.git/internal/subscription"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Subscription Service
// @version 1.0
// @description REST API for managing user subscriptions
// @host localhost:8080
func main() {

	_ = godotenv.Load("../.env", ".env")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	database, err := db.New(ctx, db.Config{
		URL:             cfg.DB.DSN(),
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	defer database.Close()

	if err := migrate.Up(ctx, database); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	appLogger := logger.New(cfg.Log.Level)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger(appLogger))

	router.GET("/hello", func(c *gin.Context) {
		c.String(200, "Hello, ahmed. this for testing !")
	})

	subRepo := subscription.NewRepository(database, appLogger)
	subService := subscription.NewService(subRepo)
	subHandler := subscription.NewHandler(subService, appLogger)
	subHandler.RegisterRoutes(router)

	docs.SwaggerInfo.Host = cfg.Swagger.Host
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	srv := &http.Server{
		Addr:    ":" + cfg.App.Port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLogger.Error("http server error", "err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("graceful shutdown failed", "err", err)
	}

	fmt.Println("Server gracefully stopped")
}
