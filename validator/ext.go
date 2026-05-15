package validator

import (
	"slices"
	"strings"
	"unicode/utf8"
)

// Define a new ExtValidator struct which contains a map of validation error messages
// for our form fields.
type ExtValidator struct {
	FieldErrors map[string]string
}

// Valid() returns true if the FieldErrors map doesn't contain any entries.
func (v *ExtValidator) Valid() bool {
	return len(v.FieldErrors) == 0
}

// AddFieldError() adds an error message to the FieldErrors map (so long as no
// entry already exists for the given key).
func (v *ExtValidator) AddFieldError(key, message string) {
	// Note: We need to initialize the map first, if it isn't already
	// initialized.
	if v.FieldErrors == nil {
		v.FieldErrors = make(map[string]string)
	}
	if _, exists := v.FieldErrors[key]; !exists {
		v.FieldErrors[key] = message
	}
}

// CheckField() adds an error message to the FieldErrors map only if a
// validation check is not 'ok'.
func (v *ExtValidator) CheckField(ok bool, key, message string) {
	if !ok {
		v.AddFieldError(key, message)
	}
}

// NotBlank() returns true if a value is not an empty string.
func NotBlank(value string) bool {
	return strings.TrimSpace(value) != ""
}

// MaxChars() returns true if a value contains no more than n characters.
func MaxChars(value string, n int) bool {
	return utf8.RuneCountInString(value) <= n
}

// MaxChars() returns true if a value contains no more than n characters.
func MinChars(value string, n int) bool {
	return utf8.RuneCountInString(value) >= n
}

// PermittedValue() returns true if a value is in a list of specific permitted
// values.
func PermittedValue[T comparable](value T, permittedValues ...T) bool {
	return slices.Contains(permittedValues, value)
}

func StringSizeBetween(value string, nMin int, nMax int) bool {
	return MinChars(value, nMin) && MaxChars(value, nMax)
}

func StringMustContain(value string, among string, times int) bool {
	countTimes := 0
	for _, vB := range []rune(among) {
		for _, vV := range []rune(value) {
			if vB == vV {
				countTimes++
			}
		}
	}
	return countTimes >= times
}

func StringInStringArray(target string, strings []string) bool {
	return slices.Contains(strings, target)
}
