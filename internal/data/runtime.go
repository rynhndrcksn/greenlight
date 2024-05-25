package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Runtime is a custom type to be used in our Movie struct.
type Runtime int32

// ErrInvalidRuntimeFormat defines a custom error for UnmarshalJSON to return.
var ErrInvalidRuntimeFormat = errors.New("invalid runtime format")

// MarshalJSON method satisfies the json.Marshaler interface.
// This means we can use this to customize how "Runtime" in our Movie struct looks.
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)
	quotedJSONValue := strconv.Quote(jsonValue)
	return []byte(quotedJSONValue), nil
}

// UnmarshalJSON method satisfied the json.Unmarshal interface.
// This lets us accept a format of "<runtime> mins".
func (r *Runtime) UnmarshalJSON(jsonValue []byte) error {
	// Remove the quotes around our string.
	unquotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	// Split the string to isolate the part containing the number.
	parts := strings.Split(unquotedJSONValue, " ")

	// Sanity check that the string is in the expected format.
	if len(parts) != 2 || parts[1] != "mins" {
		return ErrInvalidRuntimeFormat
	}

	// Parse the string containing the number into an int32.
	i, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	// Convert the int32 into a Runtime type and assign this to the receiver.
	// Note that we use the * operator to dereference the receiver (which is a pointer
	// to the Runtime type) in order to set the underlying value of the pointer.
	*r = Runtime(i)

	return nil
}
