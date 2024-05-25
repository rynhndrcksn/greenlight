package validator

import (
	"regexp"
	"slices"
)

var (
	// EmailRX contains the regular expression for checking email addresses.
	// This expression is taken from: https://html.spec.whatwg.org/#valid-e-mail-address.
	EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
)

// Validator struct contains a map of all the validation errors.
type Validator struct {
	Errors map[string]string
}

// New returns a new Validator instance.
func New() *Validator {
	return &Validator{
		Errors: make(map[string]string),
	}
}

// Valid returns true if there are no errors in our validator, otherwise false.
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// AddError adds an error message to the map (so long as the provided key hasn't been used yet).
func (v *Validator) AddError(key, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

// Check adds an error message to the Validator if a validation check is not "ok".
func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}

// PermittedValue returns true if a specific value is in the list of permitted values, otherwise false.
func PermittedValue[T comparable](value T, permittedValue ...T) bool {
	return slices.Contains(permittedValue, value)
}

// Matches returns true if the provided string matches a regexp pattern, otherwise false.
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

// Unique returns true if all values in a slice are unique, otherwise false.
func Unique[T comparable](values []T) bool {
	uniqueValues := make(map[T]bool)
	for _, value := range values {
		uniqueValues[value] = true
	}
	return len(uniqueValues) == len(values)
}
