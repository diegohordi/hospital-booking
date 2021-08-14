package main

import (
	"context"
	"flag"
	"fmt"
	"hospital-booking/internal/auth"
	"hospital-booking/internal/calendar"
	"hospital-booking/internal/configs"
	"hospital-booking/internal/database"
	"hospital-booking/internal/logging"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var configPath = flag.String("config", "", "Config file path")

// loadConfigurations loads system configurations based on the given config file.
func loadConfigurations() configs.Config {
	if *configPath == "" {
		log.Fatal("no config file path was given")
	}
	config, err := configs.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	return config
}

// createDBConnection creates a new database connection based on the given configuration.
func createDBConnection(config configs.Config) database.Connection {
	dbConn, err := database.NewConnection(config)
	if err != nil {
		log.Fatal(err)
	}
	return dbConn
}

func main() {
	// Load dependencies
	flag.Parse()
	config := loadConfigurations()
	dbConn := createDBConnection(config)

	// Init Authorizer service
	authorizer := auth.NewService(config, dbConn)

	// Init error logger
	logger := log.New(os.Stdout, "", log.LstdFlags)

	// Setup the HTTP router
	router := chi.NewRouter()
	router.Use(middleware.Heartbeat("/health"))
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.SetHeader("Content-type", "application/json"))

	// Setup Auth routes
	auth.Setup(router, logger, config, dbConn)

	// Setup Calendar routes
	calendar.Setup(router, logger, authorizer, config, dbConn)

	// Creates the HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.ServerPort()),
		Handler:      router,
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	// Channel to listen OS signalling in order to gracefully shutdown the HTTP server and other resources
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Starts the server
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	logging.PrintlnInfo(logger, fmt.Sprint("server started listing at ", config.ServerPort()))

	// Listens until server stop
	<-exit
	logging.PrintlnWarn(logger, "server stopped")

	// Creates a timeout to handle resources release
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		dbConn.Close()
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal(fmt.Errorf("an error occurred while server is shutting down: %w", err))
	}

	logging.PrintlnInfo(logger, "server shutdown successfully")
}
