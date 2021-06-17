package utils

import (
	"fmt"
	"runtime"
	"time"

	"github.com/dustin/go-humanize"
)

// PrintMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
// Adopted from: https://golangcode.com/print-the-current-memory-usage/
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Print("Alloc = ", humanize.Bytes(m.Alloc))
	fmt.Print("\tTotalAlloc = ", humanize.Bytes(m.TotalAlloc))
	fmt.Print("\tSys = ", humanize.Bytes(m.Sys))
	fmt.Print("\tGCSys = ", humanize.Bytes(m.GCSys))
	fmt.Print("\tNumGC = ", m.NumGC)
	fmt.Println("\tTimeGC =", m.PauseTotalNs/uint64(time.Millisecond), "ms")
}
