package utils

// Pointer is a generic function that returns a pointer to a given value of any type
func Pointer[T any](value T) *T {
	return &value
}
