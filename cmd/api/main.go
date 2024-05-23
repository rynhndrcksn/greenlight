package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// Hardcoded API version number, will later swap this out to be dynamic.
const version = "1.0.0"

// Config struct that contains all our project configurations.
type config struct {
	port int
	env  string
}

// Application struct that contains stuff we will want to use throughout out project.
type application struct {
	config config
	logger *slog.Logger
}

func main() {
	// Initialize a new config struct.
	var conf config
	flag.IntVar(&conf.port, "port", 4000, "API server port")
	flag.StringVar(&conf.env, "env", "development", "Environment (development|staging|production)")
	flag.Parse()

	// Initialize a new structured logger that writes to stdout.
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Initialize a new application.
	app := &application{
		config: conf,
		logger: logger,
	}

	// Initialize HTTP server using some sensible timeout settings.
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", conf.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// Start server.
	logger.Info("Starting server...", slog.String("addr", srv.Addr), slog.String("env", conf.env))
	err := srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
}
