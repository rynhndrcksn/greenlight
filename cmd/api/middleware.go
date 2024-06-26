package main

import (
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tomasen/realip"
	"golang.org/x/time/rate"

	"github.com/rynhndrcksn/greenlight/internal/data"
	"github.com/rynhndrcksn/greenlight/internal/validator"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a deferred function (which will always be run in the event of a panic
		// as Go unwinds the stack).
		defer func() {
			// Use the builtin recover function to check if there has been a panic or
			// not.
			if err := recover(); err != nil {
				// If there was a panic, set a "Connection: close" header on the
				// response. This acts as a trigger to make Go's HTTP server
				// automatically close the current connection after a response has been
				// sent.
				w.Header().Set("Connection", "close")
				// The value returned by recover() has the type any, so we use
				// fmt.Errorf() to normalize it into an error and call our
				// serverErrorResponse() helper. In turn, this will log the error using
				// our custom Logger type at the ERROR level and send the client a 500
				// Internal Server Error response.
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) rateLimit(next http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	// Declare a mutex and a map to hold the clients' IP addresses and rate limiters.
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// Launch a background goroutine which removes old entries from the clients map once
	// every minute.
	go func() {
		for {
			time.Sleep(time.Minute)

			// Lock the mutex to prevent any rate limiter checks from happening while
			// the cleanup is taking place.
			mu.Lock()

			// Loop through all clients.
			// If they haven't been seen within the last five minutes, delete the corresponding entry from the map.
			for ip, c := range clients {
				if time.Since(c.lastSeen) > 5*time.Minute {
					delete(clients, ip)
				}
			}

			// Importantly, unlock the mutex when the cleanup is complete.
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only carry out the check if rate limited is enabled.
		if app.config.limiter.enabled {
			// Extract the client's IP address from the request.
			ip := realip.FromRequest(r)

			// Lock the mutex to prevent this code from being executed concurrently.
			mu.Lock()

			// Check to see if the IP address already exists in the map.
			// If it doesn't, then initialize and add a new client struct to the map.
			if _, found := clients[ip]; !found {
				clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst)}
			}

			// Update the "last seen" time for the client.
			clients[ip].lastSeen = time.Now()

			// Call the Allow() method on the rate limiter for the current IP address.
			// If the request isn't allowed, unlock the mutex and send a 429 Too Many Requests response.
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				app.rateLimitExceededResponse(w, r)
				return
			}

			// Very importantly, unlock the mutex before calling the next handler in the chain.
			// Notice that we DON'T use defer to unlock the mutex, as that would mean
			// that the mutex isn't unlocked until all the handlers downstream of this
			// middleware have also returned.
			mu.Unlock()
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add the "Vary: Authorization" header to the response.
		// This indicates to any caches that the response may vary
		// based on the value of the Authorization header in the request.
		w.Header().Add("Vary", "Authorization")

		// Retrieve the value of the Authorization header from the request.
		// This will return the empty string "" if there is no such header found.
		authorizationHeader := r.Header.Get("Authorization")

		// If there is no Authorization header found, use the contextSetUser() helper
		// that we just made to add the AnonymousUser to the request context.
		// Then we call the next handler in the chain and return without
		// executing any of the code below.
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// Otherwise, we expect the value of the Authorization header to be in the format
		// "Bearer <token>". We try to split this into its constituent parts, and if the
		// header isn't in the expected format we return a 401 Unauthorized response
		// using the invalidAuthenticationTokenResponse() helper.
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Extract the actual authentication token from the header parts.
		token := headerParts[1]

		// Validate the token to make sure it is in a sensible format.
		v := validator.New()

		// If the token isn't valid, use the invalidAuthenticationTokenResponse()
		// helper to send a response, rather than the failedValidationResponse() helper
		// that we'd normally use.
		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Retrieve the details of the user associated with the authentication token,
		// again calling the invalidAuthenticationTokenResponse() helper if no
		// matching record was found.
		// IMPORTANT: Notice that we are using ScopeAuthentication as the first parameter here.
		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		// Call the contextSetUser() helper to add the user information to the request
		// context.
		r = app.contextSetUser(r, user)

		// Call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}

//func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		// Use the contextGetUser() helper that we made to retrieve the user
//		// information from the request context.
//		user := app.contextGetUser(r)
//
//		// If the user is anonymous, then call the authenticationRequiredResponse() to
//		// inform the client that they should authenticate before trying again.
//		if user.IsAnonymous() {
//			app.authenticationRequiredResponse(w, r)
//			return
//		}
//
//		// If the user is not activated, use the inactiveAccountResponse() helper to
//		// inform them that they need to activate their account.
//		if !user.Activated {
//			app.inactiveAccountResponse(w, r)
//			return
//		}
//
//		// Call the next handler in the chain.
//		next.ServeHTTP(w, r)
//	})
//}

// Create a new requireAuthenticatedUser() middleware to check that a user is not anonymous.
func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Checks that a user is both authenticated and activated.
func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	// Rather than returning this http.HandlerFunc we assign it to the variable fn.
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		// Check that a user is activated.
		if !user.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	// Wrap fn with the requireAuthenticatedUser() middleware before returning it.
	return app.requireAuthenticatedUser(fn)
}

// requirePermissions will ensure users have the appropriate permissions to access a resource.
// requirePermissions wraps around requireActivatedUser, which wraps around requireAuthenticatedUser,
// which means that when calling requirePermissions there's three checks happening.
func (app *application) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// Retrieve the user from the request context.
		user := app.contextGetUser(r)

		// Get the slice of permissions for the user.
		permissions, err := app.models.Permissions.GetAllForUser(user.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		// Check if the slice includes the required permission.
		// If it doesn't, then return a 403 forbidden response.
		if !permissions.Include(code) {
			app.notPermittedResponse(w, r)
			return
		}

		// Otherwise, they have the required permission, so we can call the next handler in the chain.
		next.ServeHTTP(w, r)
	}

	// Wrap this with the requireActivatedUser middleware before returning it.
	return app.requireActivatedUser(fn)
}

// enableCORS allows requests from all origins.
func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add the "Vary: Origin" header.
		w.Header().Add("Vary", "Origin")

		// Add the "Vary: Access-Control-Request-Method" header.
		w.Header().Add("Vary", "Access-Control-Request-Method")

		// Get the value of the request's Origin header.
		origin := r.Header.Get("Origin")

		// Only run this if there's an Origin request header present.
		if origin != "" {
			// Loop through the list of trusted origins, checking to see if the request
			// origin exactly matches one of them.
			// If there are no trusted origins, the loop won't be iterated.
			for i := range app.config.cors.trustedOrigins {
				if origin == app.config.cors.trustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)

					// Check if the request has the HTTP method OPTIONS and contains the "Access-Control-Request-Method" header.
					// If it does, then we treat it as a preflight request.
					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						// Set the necessary preflight response headers, as discussed previously.
						w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

						// Write the headers along with a 200 OK status and return from
						// the middleware with no further action.
						w.WriteHeader(http.StatusOK)
						return
					}

					break
				}
			}
		}

		// Call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}

// metricsResponseWriter type wraps an existing http.ResponseWriter and also
// contains a field for recording the response status code, and a boolean flag to
// indicate whether the response headers have already been written.
type metricsResponseWriter struct {
	wrapped       http.ResponseWriter
	statusCode    int
	headerWritten bool
}

// newMetricsResponseWriter returns a new instance of a metricsResponseWriter,
// which wraps a given http.ResponseWriter and has a status code of 200
// (which is the status code that Go will send in a HTTP response by default).
func newMetricsResponseWriter(w http.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{
		wrapped:    w,
		statusCode: http.StatusOK,
	}
}

// Header is a simple 'pass through' to the Header() method of the
// wrapped http.ResponseWriter.
func (mw *metricsResponseWriter) Header() http.Header {
	return mw.wrapped.Header()
}

// WriteHeader does a 'pass through' to the WriteHeader()
// method of the wrapped http.ResponseWriter.
// But after this returns, we also record the response status code
// (if it hasn't already been recorded) and set the headerWritten field
// to true to indicate that the HTTP response headers have now been written.
func (mw *metricsResponseWriter) WriteHeader(statusCode int) {
	mw.wrapped.WriteHeader(statusCode)

	if !mw.headerWritten {
		mw.statusCode = statusCode
		mw.headerWritten = true
	}
}

// Write does a 'pass through' to the Write() method of the
// wrapped http.ResponseWriter.
// Calling this will automatically write any response headers,
// so we set the headerWritten field to true.
func (mw *metricsResponseWriter) Write(b []byte) (int, error) {
	mw.headerWritten = true
	return mw.wrapped.Write(b)
}

// Unwrap returns the existing wrapped http.ResponseWriter.
func (mw *metricsResponseWriter) Unwrap() http.ResponseWriter {
	return mw.wrapped
}

// metrics records request-level metrics for our application.
func (app *application) metrics(next http.Handler) http.Handler {
	// Initialize the new expvar variables when the middleware chain is first built
	var (
		totalRequestsReceived           = expvar.NewInt("total_requests_received")
		totalResponsesSent              = expvar.NewInt("total_responses_sent")
		totalProcessingTimeMicroseconds = expvar.NewInt("total_processing_time_μs")
		totalResponsesSentByStatus      = expvar.NewMap("total_responses_sent_by_status")
	)

	// The following code will be run on every request.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		totalRequestsReceived.Add(1)

		// Create a new metricsResponseWriter, which wraps the original http.ResponseWriter
		mw := newMetricsResponseWriter(w)

		// Normally we would pass the normal http.ResponseWriter.
		// However, we are passing in our custom one.
		next.ServeHTTP(mw, r)

		// On the way back up the middleware chain...
		totalResponsesSent.Add(1)
		// Pull out the statusCode on our custom metricsResponseWriter.
		totalResponsesSentByStatus.Add(strconv.Itoa(mw.statusCode), 1)
		duration := time.Since(start).Microseconds()
		totalProcessingTimeMicroseconds.Add(duration)
	})
}
