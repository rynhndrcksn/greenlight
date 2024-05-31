package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"

	"github.com/rynhndrcksn/greenlight/internal/validator"
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
	// In most applications, this isn't a concern. The differences equate to a few thousandths of a millisecond but improves readability.
	// If the API is resource-constrained or handles EXTREMELY high levels of traffic, then Marshal might be better.
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	// Append a newline to make it look better in the terminal.
	js = append(js, '\n')

	// At this point, we know we won't encounter any more errors, so we can safely loop over the headers and add them.
	for key, value := range headers {
		w.Header()[key] = value
	}

	// Set extra headers and send response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

// readJSON is a helper for reading JSON requests.
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Use http.MaxBytesReader() to limit the size of the request body to 1MB.
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	// Initialize json.Decoder and call the DisallowUnknownFields() method on it before decoding.
	// Now any field that can't be mapped to the target destination raises an error.
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	// Decode the request body into the target destination.
	err := dec.Decode(dst)
	if err != nil {
		// If there's an error, we need to triage the type of error
		// and return the right type of response...
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		// Not properly formatted JSON.
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		// In some cases Decode() might return an io.ErrUnexpectedEOF for syntax errors.
		// There is an open issues regarding this: https://github.com/golang/go/issues/25956.
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		// A json.UnmarshalTypeError occurs when the JSON value is the wrong type for target destination.
		// If the error relates to a specific field, we include that for easier client debugging.
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		// The body is empty.
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		// JSON contains a field that can't be mapped to target destination.
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		// JSON is larger than our defined max bytes.
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger that %d bytes", maxBytesError.Limit)

		// If a non-nil pointer is passed to Decode(), we get an json.invalidUnmarshalError.
		// In the event this happens, panicking is better than returning an error.
		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		// For anything else, return the standard error.
		default:
			return err
		}
	}
	// Make sure only 1 JSON body was sent.
	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}
	return nil
}

// readString is a helper method for returning values from a query string or
// a provided default if no matching query string was found.
func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	// Extract the value from the query string.
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	return s
}

// readCSV is a helper method for returning comma separated values from a query string
func (app *application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	// Extract the values from the query string.
	csv := qs.Get(key)

	if csv == "" {
		return defaultValue
	}

	return strings.Split(csv, ",")
}

// readInt is a helper method for returning integers from a query string.
func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	return i
}

// background helper accepts an arbitrary function as a parameter.
func (app *application) background(fn func()) {
	app.wg.Add(1)
	// Launch a background goroutine.
	go func() {
		// Use "defer" to decrement the WaitGroup counter right before the goroutine returns.
		defer app.wg.Done()

		// Recover any panics that arise.
		defer func() {
			if err := recover(); err != nil {
				app.logger.Error(fmt.Sprintf("%v%", err))
			}
		}()

		// Execute our arbitrary function that was passed as the parameter.
		fn()
	}()
}
