// package algorithms provides generified map/filter/reduce functions.
package algorithms

// Map applies the function f to each element of the slice and returns a new slice containing the results.
func Map[T, R any](s []T, f func(T) R) []R {
	r := make([]R, 0, len(s))
	for _, v := range s {
		r = append(r, f(v))
	}
	return r
}

// Filter returns a new slice containing all elements of the slice that satisfy the predicate function.
func Filter[T any](s []T, f func(T) bool) []T {
	r := make([]T, 0, len(s))
	for _, v := range s {
		if f(v) {
			r = append(r, v)
		}
	}
	return r
}
