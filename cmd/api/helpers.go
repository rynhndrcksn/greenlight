package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

func (app *application) readIdParam(r *http.Request) (int64, error) {
	// When httprouter is parsing a request, any interpolated URL parameters will be
	// stored in the request context. We can use the ParamsFromContext() function to
	// retrieve a slice containing these parameter names and values.
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil {
		return 0, errors.New("invalid id parameter")
	}

	return id, nil
}

// Define an envelope type for wrapping JSON responses in.
type envelope map[string]any

// writeJSON is a helper for sending JSON responses.
func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	// Marshal our data.
	// Note: MarshalIndent generally runs ~65% slower, uses ~30% more memory, and makes 2 more heap allocations than Marshal.
	// In most applications this isn't a concern. The differences equate to a few thousandths of a millisecond but improves readability.
	// If the API is resource-constrained or handles EXTREMELY high levels of traffic, then Marshal might be better.
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	// Append a newline to make it look better in the terminal.
	js = append(js, '\n')

	// At this point we know we won't encounter any more errors, so we can safely loop over the headers and add them.
	for key, value := range headers {
		w.Header()[key] = value
	}

	// Set extra headers and send response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}
