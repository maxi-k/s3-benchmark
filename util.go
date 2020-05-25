package main

import (
	"fmt"
	"runtime"
	"strings"
)

// returns an object size iterator, starting from 1 KB and double in size by each iteration
func payloadSizeGenerator() func() uint64 {
	nextPayloadSize := uint64(payloadsMin)

	return func() uint64 {
		thisPayloadSize := nextPayloadSize
		nextPayloadSize *= 2
		return thisPayloadSize
	}
}

// adjust the sample count for small instances and for low thread counts (so that the test doesn't take forever)
func getTargetSampleCount(threads, tasks int) int {
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

func getHardwareConfig() (int, int) {
	hwThreads := runtime.NumCPU()
	hwCores := minimumOf(hwThreads/2, 1) // assume hyperthreading
	return hwCores, hwThreads
}

// go doesn't seem to have a min function in the std lib!
func minimumOf(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// formats bytes to KB or MB
func byteFormat(bytes float64) string {
	if bytes >= 1024*1024 {
		return fmt.Sprintf("%.f MB", bytes/1024/1024)
	}
	return fmt.Sprintf("%.f KB", bytes/1024)
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
