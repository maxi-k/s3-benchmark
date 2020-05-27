package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"strings"
)

// returns an object size iterator, starting from 1 MB and double in size by each iteration
func payloadSizeGenerator() func() usize {
	nextPayloadSize := usize(payloadsMin * unitMB)

	return func() usize {
		thisPayloadSize := nextPayloadSize
		nextPayloadSize *= 2
		return thisPayloadSize
	}
}

// adjust the sample count for small instances and for low thread counts (so that the test doesn't take forever)
func getTargetSampleCount(threads, tasks usize) usize {
	if instanceType == "" {
		return minimumOf(50, tasks)
	}
	if !strings.Contains(instanceType, "xlarge") && !strings.Contains(instanceType, "metal") {
		return minimumOf(50, tasks)
	}
	if threads <= 4 {
		return minimumOf(100, tasks)
	}
	if threads <= 8 {
		return minimumOf(250, tasks)
	}
	if threads <= 16 {
		return minimumOf(500, tasks)
	}
	return tasks
}

func getHardwareConfig() (usize, usize) {
	hwThreads := usize(runtime.NumCPU())
	hwCores := minimumOf(hwThreads/2, 1) // assume hyperthreading
	return hwCores, hwThreads
}

// go doesn't seem to have a min function in the std lib!
func minimumOf(x, y usize) usize {
	if x < y {
		return x
	}
	return y
}

// formats bytes to KB, MB or GB
func byteFormat(bytes float64) string {
	if bytes >= unitGB {
		return fmt.Sprintf("%.f GB", bytes/unitGB)
	}
	if bytes >= unitMB {
		return fmt.Sprintf("%.f MB", bytes/unitMB)
	}
	return fmt.Sprintf("%.f KB", bytes/unitKB)
}

// represents a byte range to fetch from an s3 object
type byteRange struct {
	start usize
	end   usize
}

// returns a random byte range with `chunkSize` inside `[0, size]`
func randomByteRange(size usize, chunkSize usize) byteRange {
	if size < chunkSize {
		fmt.Printf("Error: %d < %d\n", size, chunkSize)
		panic("ChunkSize was larger than overall Size!")
	}
	offset := usize(rand.Intn(int(size - chunkSize + 1)))
	return byteRange{offset, offset + chunkSize}
}

// returns a http range header from the given byte range
func (r *byteRange) ToHTTPHeader() string {
	return fmt.Sprintf("bytes=%d-%d", r.start, r.end)
}

func (r *byteRange) Size() usize {
	return r.end - r.start
}

// comparator to sort by first byte latency
type ByFirstByte []latency

func (a ByFirstByte) Len() int           { return len(a) }
func (a ByFirstByte) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByFirstByte) Less(i, j int) bool { return a[i].FirstByte < a[j].FirstByte }

// comparator to sort by last byte latency
type ByLastByte []latency

func (a ByLastByte) Len() int           { return len(a) }
func (a ByLastByte) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLastByte) Less(i, j int) bool { return a[i].LastByte < a[j].LastByte }
