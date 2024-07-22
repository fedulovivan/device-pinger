package utils

func Truncate(in []byte, limit uint) []byte {
	if uint(len(in)) <= limit {
		return in
	}
	return in[:limit]
}
