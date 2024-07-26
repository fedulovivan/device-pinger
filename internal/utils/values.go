package utils

// return values from given map as a slice
// https://stackoverflow.com/a/71635953/1012298
func Values[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}
