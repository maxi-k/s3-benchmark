package main

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"strings"
)

/*****
 *  INSTANCE AND PAYLOAD UTILITIES
 *****/

// returns an object size iterator, starting from 1 MB and double in size by each iteration
func payloadSizeGenerator() func() (usize, bool) {
	if payloadsReverse {
		nextPayloadSize := usize(payloadsMax * unitMB)
		minPayloadSize := payloadsMin * unitMB
		return func() (usize, bool) {
			thisPayloadSize := nextPayloadSize
			nextPayloadSize /= payloadsStep
			return thisPayloadSize, thisPayloadSize >= minPayloadSize
		}
	} else {
		nextPayloadSize := usize(payloadsMin * unitMB)
		maxPayloadSize := payloadsMax * unitMB
		return func() (usize, bool) {
			thisPayloadSize := nextPayloadSize
			nextPayloadSize *= payloadsStep
			return thisPayloadSize, thisPayloadSize <= maxPayloadSize
		}
	}
}

func threadCountGenerator(min usize) func() usize {
	nextThreadCount := usize(min)
	if threadsStep > 1 {
		return func() usize {
			thisThreadCount := nextThreadCount
			nextThreadCount = uint64(threadsStep * float64(nextThreadCount))
			return thisThreadCount
		}
	} else {

		return func() usize {
			thisThreadCount := nextThreadCount
			nextThreadCount += uint64(math.Abs(threadsStep))
			return thisThreadCount
		}
	}
}

func getMinMaxThreadCount(staticCount bool, minArg float64, maxArg float64) (usize, usize) {
	if staticCount {
		if !isWhole(minArg) || !isWhole(maxArg) {
			panic("Passed float values to threads-min or threads-max with threads-static.")
		}
		return roundToUsize(minArg), roundToUsize(maxArg)
	} else {
		_, hwThreads := getHardwareConfig()
		return roundToUsize(float64(hwThreads) * minArg), roundToUsize(float64(hwThreads) * maxArg)
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
	return minimumOf(threads*tasks, runSampleCap)
}

func getHardwareConfig() (usize, usize) {
	hwThreads := usize(runtime.NumCPU())
	hwCores := maximumOf(hwThreads/2, 1) // assume hyperthreading
	return hwCores, hwThreads
}

/*****
 *  MATH UTILITIES
 *****/

// go doesn't seem to have a min function in the std lib!
func minimumOf(x, y usize) usize {
	if x < y {
		return x
	}
	return y
}

// go doesn't seem to have a min function in the std lib!
func maximumOf(x, y usize) usize {
	if x > y {
		return x
	}
	return y
}

func roundToUsize(v float64) usize {
	return uint64(math.Round(v))
}

func isWhole(v float64) bool {
	return v == math.Trunc(v)
}

/*****
 *  BYTE UTILITIES
 *****/

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

// formats bytes to KB, MB or GB
func byteFormat(bytes float64) string {
	if bytes >= unitGB {
		return fmt.Sprintf("%.f GiB", bytes/unitGB)
	}
	if bytes >= unitMB {
		return fmt.Sprintf("%.f MiB", bytes/unitMB)
	}
	return fmt.Sprintf("%.f KiB", bytes/unitKB)
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

/*****
 *  PRINT UTILITIES
 *****/

const printEOL = "\033[59G|"

func printIntVar(name string, value usize) {
	fmt.Printf("| \033[1m%s\033[0m\t:\t%d%s\n", name, value, printEOL)
}

func printBoolVar(name string, value bool) {
	s := "FALSE"
	if value {
		s = "TRUE"
	}
	fmt.Printf("| \033[1m%s\033[0m\t:\t%s%s\n", name, s, printEOL)
}

func printStrVar(name string, value string) {
	fmt.Printf("| \033[1m%s\033[0m\t:\t%s%s\n", name, value, printEOL)
}

func printConfiguration() {
	fmt.Printf("\n+------------------- \033[1;32mRUN CONFIGURATION\033[0m -------------------+\n")

	printBoolVar("Dry Run?", dryRun)
	printStrVar("EC2 Region", region)
	printStrVar("Instance Type", instanceType)
	printStrVar("Bucket Name", bucketName)
	printStrVar("Object Name", objectName)
	printIntVar("Payloads Min", payloadsMin)
	printIntVar("Payloads Max", payloadsMax)
	printIntVar("Threads Min", threadsMin)
	printIntVar("Threads Max", threadsMax)
	endStr := "+---------------------------------------------------------+\n"
	fmt.Print(endStr)

	fmt.Printf("\n+------------------- \033[1;32mDETECTED HARDWARE\033[0m -------------------+\n")
	hwCores, hwThreads := getHardwareConfig()
	printIntVar("Detected Cores", hwCores)
	printIntVar("Detected HW Threads", hwThreads)
	fmt.Print(endStr)
}

func printDryRun(threadCount usize, payload usize) {
	fmt.Printf("⟶ Dry Run Request: \t \033[1m%d\033[0m \t Threads and Payload Size \t \033[1m%s\033[0m \n", threadCount, byteFormat(float64(payload)))
}
