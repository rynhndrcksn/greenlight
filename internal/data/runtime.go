package data

import (
	"fmt"
	"strconv"
)

// Runtime is a custom type to be used in our Movie struct.
type Runtime int32

// MarshalJSON method satisfies the json.Marshaler interface.
// This means we can use this to customize how "Runtime" in our Movie struct looks.
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)
	quotedJSONValue := strconv.Quote(jsonValue)
	return []byte(quotedJSONValue), nil
}
