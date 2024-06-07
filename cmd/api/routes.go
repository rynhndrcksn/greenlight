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

	// Register /v1/ routes
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodGet, "/v1/movies", app.requirePermission("movies:read", app.listMoviesHandler))
	router.HandlerFunc(http.MethodPost, "/v1/movies", app.requirePermission("movies:write", app.createMovieHandler))
	router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.requirePermission("movies:read", app.showMovieHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/movies/:id", app.requirePermission("movies:write", app.updateMovieHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/movies/:id", app.requirePermission("movies:write", app.deleteMovieHandler))
	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	return app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router))))
}
