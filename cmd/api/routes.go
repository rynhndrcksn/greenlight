package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	// Initialize new httprouter instance
	router := httprouter.New()

	// Tell httprouter to use our custom notFoundResponse handler.
	router.NotFound = http.HandlerFunc(app.notFoundResponse)

	// Tell httprouter to use our custom methodNotAllowed handler.
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	// Register routes
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodPost, "/v1/movies", app.createMovieHandler)
	router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.showMovieHandler)
	router.HandlerFunc(http.MethodPut, "/v1/movies/:id", app.updateMovieHandler)

	return app.recoverPanic(router)
}
