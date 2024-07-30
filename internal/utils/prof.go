package utils

import (
	"log/slog"
	"runtime"
)

//
// from https://gist.github.com/j33ty/79e8b736141be19687f565ea4c6f4226
//

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// For info on each, see: https://golang.org/pkg/runtime/#MemStats

	// fmt.Printf("Alloc = %v KiB", bToKb(m.Alloc))
	// fmt.Printf("\tTotalAlloc = %v KiB", bToKb(m.TotalAlloc))
	// fmt.Printf("\tSys = %v KiB", bToKb(m.Sys))
	// fmt.Printf("\tNumGC = %v\n", m.NumGC)

	slog.Debug(
		"[MAIN] Memory Usage in KiB",
		"ALLOC", bToKb(m.Alloc),
		"TOTAL_ALLOC", bToKb(m.TotalAlloc),
		"SYS", bToKb(m.Sys),
		"NUM_GC", m.NumGC,
	)
}

func bToKb(b uint64) uint64 {
	return b / 1024
}
