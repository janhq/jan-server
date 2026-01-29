package functional

// Map applies a function to each element of a slice and returns a new slice with the results
func Map[T any, U any](slice []T, fn func(T) U) []U {
	result := make([]U, len(slice))
	for i, item := range slice {
		result[i] = fn(item)
	}
	return result
}

// Filter returns a new slice containing only the elements that satisfy the predicate
func Filter[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0)
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// Reduce applies a function against an accumulator and each element in the slice to reduce it to a single value
func Reduce[T any, U any](slice []T, initial U, fn func(U, T) U) U {
	accumulator := initial
	for _, item := range slice {
		accumulator = fn(accumulator, item)
	}
	return accumulator
}

// Find returns the first element that satisfies the predicate, or the zero value if none found
func Find[T any](slice []T, predicate func(T) bool) (T, bool) {
	for _, item := range slice {
		if predicate(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
}

// Any returns true if any element in the slice satisfies the predicate
func Any[T any](slice []T, predicate func(T) bool) bool {
	for _, item := range slice {
		if predicate(item) {
			return true
		}
	}
	return false
}

// All returns true if all elements in the slice satisfy the predicate
func All[T any](slice []T, predicate func(T) bool) bool {
	for _, item := range slice {
		if !predicate(item) {
			return false
		}
	}
	return true
}
