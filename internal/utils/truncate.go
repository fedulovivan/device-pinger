package utils

// https://stackoverflow.com/questions/58635507/rune-vs-byte-ranging-over-string
func Truncate(in string, limit uint) string {
	if uint(len([]rune(in))) <= limit {
		return in
	}
	return string([]rune(in)[:limit])
}
