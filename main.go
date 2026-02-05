package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"monitoring-energy-service/docs"
	"monitoring-energy-service/internal/infrastructure/adapters/rest"
	"monitoring-energy-service/internal/infrastructure/conf"
	"monitoring-energy-service/internal/infrastructure/conf/database"
	"monitoring-energy-service/internal/infrastructure/container"

	"github.com/caarlos0/env/v11"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var (
	buildDate string
	gitCommit string
)

// @title           Monitoring Energy Service API
// @version         1.0
// @description     Go microservice template with hexagonal architecture
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:9000
// @BasePath  /

// @schemes http https
func main() {
	// Initialize structured JSON logging
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	slog.Info("starting application", "build_date", buildDate, "git_commit", gitCommit)

	err := godotenv.Load(".env")
	if err != nil {
		slog.Warn("couldn't load .env file", "error", err)
	}

	cfg := &conf.Config{}
	opts := env.Options{OnSet: conf.OnSetConfig}
	if err := env.ParseWithOptions(cfg, opts); err != nil {
		slog.Error("failed to parse config", "error", err)
		os.Exit(1)
	}

	port := cfg.Port
	environment := cfg.Env

	// Update swagger host dynamically
	docs.SwaggerInfo.Host = "localhost:" + port

	kafkaBrokers := strings.Split(cfg.ListKafkaBrokers, ",")
	consumerGroup := cfg.ConsumeGroup
	autoOffsetKafka := "latest"

	db, err := database.SetupDatabasePsql()
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	err = database.PerformMigrations(db)
	if err != nil {
		slog.Error("failed to perform migrations", "error", err)
		os.Exit(1)
	}

	err = database.SeedDB(db, "./db")
	if err != nil {
		slog.Error("failed to seed database", "error", err)
		os.Exit(1)
	}

	timeoutSeconds := cfg.HttpClientTimeout
	httpClient := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}

	c := container.NewContainer(
		db,
		kafkaBrokers,
		consumerGroup,
		httpClient,
		autoOffsetKafka,
		container.WithConfig(*cfg),
	)

	// Start Kafka consumer in background
	go c.KafkaService.ConsumeEvents()

	// Start Event Generator in background
	go c.EventGenerator.Start()

	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/healthz", "/readyz", "/swagger/*any"},
	}))
	router.Use(gin.Recovery())

	suffixesFromEnv := strings.Split(cfg.AllowedCorsSuffixes, ",")
	allowedSuffixes := make([]string, 0, len(suffixesFromEnv))
	for _, suffix := range suffixesFromEnv {
		trimmedSuffix := strings.TrimSpace(suffix)
		if trimmedSuffix != "" {
			allowedSuffixes = append(allowedSuffixes, trimmedSuffix)
		}
	}

	router.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			if strings.HasPrefix(origin, "http://localhost:") {
				return true
			}

			for _, suffix := range allowedSuffixes {
				if strings.HasSuffix(origin, suffix) {
					return true
				}
			}

			return false
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	}))

	mainGroup := router.Group("")
	rest.SetupRoutes(mainGroup, c)

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.GET("/healthz", gin.WrapF(HealthCheck))
	router.GET("/readyz", gin.WrapF(HealthCheck))

	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	if environment == "dev" {
		slog.Info("running in development mode",
			"swagger_url", "http://localhost:"+port+"/swagger/index.html",
		)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Graceful shutdown: listen for SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server in a goroutine
	go func() {
		slog.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Block until signal received
	sig := <-quit
	slog.Info("signal received, starting graceful shutdown", "signal", sig)

	// Shutdown application components
	c.Shutdown()

	// Shutdown HTTP server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	slog.Info("server exited gracefully")
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
