// package algorithms provides generified map/filter/reduce functions.
package algorithms

// Map applies the function f to each element of the slice and returns a new slice containing the results.
func Map[T, R any](s []T, f func(T) R) []R {
	var r []R
	for _, v := range s {
		r = append(r, f(v))
	}
	return r
}
