package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/rynhndrcksn/greenlight/internal/data"
)

// Hardcoded API version number, will later swap this out to be dynamic.
const version = "1.0.0"

// Config struct that contains all our project configurations.
type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
}

// Application struct that contains stuff we will want to use throughout out project.
type application struct {
	config config
	logger *slog.Logger
	models data.Models
}

func main() {
	// Initialize a new config struct.
	var conf config
	flag.IntVar(&conf.port, "port", 4000, "API server port")
	flag.StringVar(&conf.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&conf.db.dsn, "dsn", os.Getenv("GREENLIGHT_DB_DSN"), "Database DSN")
	flag.IntVar(&conf.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&conf.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.DurationVar(&conf.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")
	flag.Float64Var(&conf.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&conf.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&conf.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
	flag.Parse()

	// Initialize a new structured logger that writes to stdout.
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Initialize a new db connection
	db, err := openDB(conf)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connection pool established")

	// Initialize a new application.
	app := &application{
		config: conf,
		logger: logger,
		models: data.NewModels(db),
	}

	err = app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

// openDB returns a sql.DB connection pool.
func openDB(cfg config) (*sql.DB, error) {
	// Use sql.Open() to create an empty connection pool, using the DSN from the config
	// struct.
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	// Set the maximum number of open (in-use + idle) connections in the pool. Note that
	// passing a value less than or equal to 0 will mean there is no limit.
	db.SetMaxOpenConns(cfg.db.maxOpenConns)

	// Set the maximum number of idle connections in the pool. Again, passing a value
	// less than or equal to 0 will mean there is no limit.
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	// Set the maximum idle timeout for connections in the pool. Passing a duration less
	// than or equal to 0 will mean that connections are not closed due to their idle time.
	db.SetConnMaxIdleTime(cfg.db.maxIdleTime)

	// Create a context with a 5-second timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use PingContext() to establish a new connection to the database, passing in the
	// context we created above as a parameter. If the connection couldn't be
	// established successfully within the 5-second deadline, then this will return an
	// error. If we get this error, or any other, we close the connection pool and
	// return the error.
	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	// Return the sql.DB connection pool.
	return db, nil
}
