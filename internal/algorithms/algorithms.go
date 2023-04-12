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

// Equal returns true if all elements are equal.
func Equal[T comparable](first, second T, rest ...T) bool {
	if first != second {
		return false
	}
	if len(rest) > 0 {
		return Equal(second, rest[0], rest[1:]...)
	}
	return true
}

// Reverse reverses the order of the elements in the slice.
func Reverse[T any](a []T) {
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
}
