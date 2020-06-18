package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type usize = uint64

const u1 = uint64(1)

// represents the duration from making an S3 GetObject request to getting the first byte and last byte
type latency struct {
	FirstByte time.Duration
	LastByte  time.Duration
}

// summary statistics used to summarize first byte and last byte latencies
type stat int

const (
	min stat = iota + 1
	max
	avg
	p25
	p50
	p75
	p90
	p99
)

// a benchmark record for one object size and thread count
type benchmark struct {
	objectSize  usize
	threads     int
	firstByte   map[stat]float64
	lastByte    map[stat]float64
	dataPoints  []latency
	sampleCount usize
}

// unit definitions
const unitKB = 1024
const unitMB = unitKB * 1024
const unitGB = unitMB * 1024

// absolute limits
const maxThreads = 64

// default settings
const defaultRegion = "eu-central-1"

// default bucket name
const defaultBucketName = "masters-thesis-mk"

// default object name
const defaultObjectName = "benchmark/largefile-100G.bin"

// the hostname or EC2 instance id
var hostname = getHostname()

// the EC2 instance region if available
var region = getRegion()

// the endpoint URL if applicable
var endpoint string

// the EC2 instance type if available
var instanceType = getInstanceType()

var bucketName string
var objectName string
var objectInfo s3.HeadObjectOutput
var objectSize usize

// the min and max object sizes to test - 1 = 1 MB, and the size doubles with every increment
var payloadsMin usize
var payloadsMax usize
var payloadsStep usize
var payloadsReverse bool

// the min and max thread count to use in the test
var useStaticThreadCount bool
var threadsMin usize
var threadsMax usize
var threadsStep float64

// the number of samples to collect for each benchmark record and thread
var samples usize

// the maximum for how much samples to collect per benchmark record
var runSampleCap usize

// a test mode to find out when EC2 network throttling kicks in
var throttlingMode bool
var throttlingModeCsvInterval usize

// if not empty, the results of the test get uploaded to S3 using this key prefix
var csvResults string

// if not empty, cpu usage statistics get uploaded to S3 using this key prefix
var statResults string

// the S3 SDK client
var s3Client *s3.S3

// flag to only print what would happen, not actually do it
var dryRun bool

// flag to only print the parsed configuration
var onlyPrintConfig bool

var programEntryTime time.Time

// program entry point
func main() {
	programEntryTime = time.Now()
	// parse the program arguments and set the global variables
	parseFlags()
	if onlyPrintConfig {
		return
	}
	// set up the S3 SDK
	setupS3Client()
	// setup variables, buckets, objects etc.
	setup()
	// run the test against the uploaded data
	runBenchmark()
	// remove the objects uploaded to S3 for this test (but doesn't remove the bucket)
	cleanup()
}

func parseFlags() {
	staticThreadsArg := flag.Bool("threads-static", false,
		"If true, interprete threads-min and threads-max as static counts instead of multiples of the hardware thread count.\n"+
			"It's advised to explicitly set threads-min and threads-max if this option is given.")
	threadsMinArg := flag.Float64("threads-min", 1,
		"The minimum number of threads to use when fetching objects from S3 as a multiple of the hardware thread count.")
	threadsMaxArg := flag.Float64("threads-max", 2,
		"The maximum number of threads to use when fetching objects from S3 as a multiple of the hardware thread count.")
	threadsStepArg := flag.Float64("threads-step", 2,
		"What increase in thread count per benchmark run is. Positive means multiplicative, negative means additive.")
	payloadsMinArg := flag.Uint64("payloads-min", 10,
		"The minimum object size to test, with 1 = 1 MB, and every increment is a double of the previous value.")
	payloadsMaxArg := flag.Uint64("payloads-max", 160,
		"The maximum object size to test, with 1 = 1 MB, and every increment is a double of the previous value.")
	payloadsStepArg := flag.Uint64("payloads-step", 2,
		"What the multiplicative increase in payload size per benchmark run is (size *= step). Must be > 1")
	payloadsReverseArg := flag.Bool("payloads-reverse", false,
		"If true, start with the largest payload size first and decrease from there")
	samplesArg := flag.Uint64("samples", 100,
		"The number of samples to collect for each test of a single object size and thread count.")
	samplesCapArg := flag.Uint64("samples-cap", 7200,
		"The maximum number of samples to collect for each test of a single object size.")
	bucketNameArg := flag.String("bucket-name", defaultBucketName,
		"The name of the bucket where the test object is located")
	objectNameArg := flag.String("object-name", defaultObjectName,
		"The name of the large object file where data will be fetched from")
	regionArg := flag.String("region", defaultRegion,
		"Sets the AWS region to use for the S3 bucket. Only applies if the bucket doesn't already exist.")
	endpointArg := flag.String("endpoint", "",
		"Sets the S3 endpoint to use. Only applies to non-AWS, S3-compatible stores.")
	fullArg := flag.Bool("full", false,
		"Runs the full exhaustive test, and overrides the threads and payload arguments.")
	throttlingModeArg := flag.Bool("throttling-mode", false,
		"Runs a continuous test to find out when EC2 network throttling kicks in.")
	throttlingModeCsvIntervalArg := flag.Uint64("throttling-csv-interval", 128,
		"After how many test runs in throttling mode should the csv results be uploaded to s3?")
	csvResultsArg := flag.String("upload-csv", "",
		"Uploads the test results to S3 as a CSV file.")
	statResultsArg := flag.String("upload-stats", "",
		"Upload CPU stats from during the benchmark to S3 as a CSV file.")

	dryRunArg := flag.Bool("dry-run", false, "Makes a dry run")
	printConfigArg := flag.Bool("print", false, "Prints the parsed configuration and exits")

	// parse the arguments and set all the global variables accordingly
	flag.Parse()

	if *bucketNameArg != "" {
		bucketName = *bucketNameArg
	}

	if *objectNameArg != "" {
		objectName = *objectNameArg
	}

	if *regionArg != "" {
		region = *regionArg
	}

	if *endpointArg != "" {
		endpoint = *endpointArg
	}

	payloadsMin = *payloadsMinArg
	payloadsMax = *payloadsMaxArg
	payloadsReverse = *payloadsReverseArg
	useStaticThreadCount = *staticThreadsArg
	threadsMin, threadsMax = getMinMaxThreadCount(useStaticThreadCount, *threadsMinArg, *threadsMaxArg)
	samples = *samplesArg
	runSampleCap = *samplesCapArg
	csvResults = *csvResultsArg
	statResults = *statResultsArg
	dryRun = *dryRunArg
	onlyPrintConfig = *printConfigArg

	if payloadsMin > payloadsMax {
		payloadsMin = payloadsMax
	}

	if threadsMin > threadsMax {
		threadsMin = threadsMax
	}

	if *payloadsStepArg <= 1 {
		panic("Payload size must increase in each step, please provide -payloads-step > 1!")
	}
	payloadsStep = *payloadsStepArg

	if *threadsStepArg == 0 || *threadsStepArg == 1 {
		panic("Threads step cannot be 0 or one. Provide negative values for additive behavior, positive for multiplicative. Use -dry-run to test behavior without running.")
	}
	threadsStep = *threadsStepArg

	if *fullArg {
		// if running the full exhaustive test, the threads and payload arguments get overridden with these
		useStaticThreadCount = true
		threadsMin = 1
		threadsMax = 48
		threadsStep = 1
		payloadsMin = 1   //   1 MB
		payloadsMax = 256 // 256 MB
	}

	if *throttlingModeArg {
		// if running the network throttling test, the threads and payload arguments get overridden with these
		useStaticThreadCount = false
		threadsMin = 1
		threadsMax = 1
		threadsStep = 1
		payloadsMin = 20 // 10 MB
		payloadsMax = 20 // 10 MB
		throttlingMode = *throttlingModeArg
	}
	throttlingModeCsvInterval = *throttlingModeCsvIntervalArg

	printConfiguration()
}

func setupS3Client() {
	// gets the AWS credentials from the default file or from the EC2 instance profile
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic("Unable to load AWS SDK config: " + err.Error())
	}

	// set the SDK region to either the one from the program arguments or else to the same region as the EC2 instance
	cfg.Region = region

	// set the endpoint in the configuration
	if endpoint != "" {
		cfg.EndpointResolver = aws.ResolveWithEndpointURL(endpoint)
	}

	// set a 3-minute timeout for all S3 calls, including downloading the body
	cfg.HTTPClient = &http.Client{
		Timeout: time.Second * 180,
	}

	// crete the S3 client
	s3Client = s3.New(cfg)

	// custom endpoints don't generally work with the bucket in the host prefix
	if endpoint != "" {
		s3Client.ForcePathStyle = true
	}
}

func setup() {
	fmt.Print("\n---------- \033[1;32mSETUP\033[0m ----------\n\n")
	fmt.Print("--- Fetching object size ---\n")
	fmt.Printf("Bucket: %s \n Object: %s \n", bucketName, objectName)

	objReq := s3Client.HeadObjectRequest(&s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    &objectName,
	})
	objRes, err := objReq.Send()
	if err != nil {
		panic("Failed to get object head" + err.Error())
	}

	objectInfo = *objRes
	objectSize = usize(*objRes.ContentLength)

	fmt.Print("\n")
}

func runBenchmark() {
	fmt.Print("\n---------- \033[1;32mBENCHMARK\033[0m ----------\n\n")

	// array of csv records used to upload the results to S3 when the test is finished
	var csvRecords [][]string
	// array of cpu stat measurements to upload to s3 when the test is finished
	var csvStats [][]string

	// an object size iterator that starts from 1 MB and doubles the size on every iteration
	payloadIter := payloadSizeGenerator()

	// loop over every payload size
	for payload, hasNext := payloadIter(); hasNext; payload, hasNext = payloadIter() {

		// print the header for the benchmark of this object size
		if !dryRun {
			printHeader(payload)
		}

		// extract thread count behavior to generator function
		nextThreadCount := threadCountGenerator(threadsMin)

		// run a test per thread count and object size combination
		for t := nextThreadCount(); t <= threadsMax; t = nextThreadCount() {
			// if throttling mode, loop forever
			for n := u1; true; n++ {
				if !dryRun {
					recordLines, statLines := execTest(t, payload, n)
					csvRecords = append(csvRecords, recordLines)
					csvStats = append(csvStats, statLines)
				} else {
					printDryRun(t, payload)
				}
				if throttlingMode {
					if n%throttlingModeCsvInterval == 0 {
						timeStr := time.Now().Format("2006-01-02--15-04-05")
						nStr := fmt.Sprintf("%d", n)
						if csvResults != "" {
							filename := "results/" + csvResults + "-" + nStr + "-@" + timeStr + "@-" + instanceType
							if dryRun {
								fmt.Printf("Would upload file %s with %d rows to s3\n", filename, len(csvRecords))
							} else {
								uploadCsv(filename, csvRecords)
							}
						}

						if statResults != "" {
							filename := "stats/" + statResults + "-" + nStr + "-@" + timeStr + "@-" + instanceType
							if dryRun {
								fmt.Printf("Would upload file %s with %d rows to s3\n", filename, len(csvStats))
							} else {
								uploadCsv(filename, csvStats)
							}
						}
						csvRecords = nil
						csvStats = nil
						if dryRun {
							fmt.Println("Uploaded and cleared records...")
						}
					}
				} else {
					break
				}
			}
		}
		if !dryRun {
			fmt.Print("+---------+----------------+------------------------------------------------+------------------------------------------------+\n\n")
		}
	}

	// if the csv option is true, upload the csv results to S3
	if csvResults != "" {
		filename := "results/" + csvResults + "-" + instanceType
		if dryRun {
			fmt.Printf("Would upload file %s with %d rows to s3\n", filename, len(csvRecords))
		} else {
			uploadCsv(filename, csvRecords)
		}
	}

	if statResults != "" {
		filename := "stats/" + statResults + "-" + instanceType
		if dryRun {
			fmt.Printf("Would upload file %s with %d rows to s3\n", filename, len(csvStats))
		} else {
			uploadCsv(filename, csvStats)
		}
	}
}

func execTest(threadCount usize, payloadSize usize, runNumber usize) ([]string, []string) {
	// this overrides the sample count on small hosts that can get overwhelmed by a large throughput
	samples := getTargetSampleCount(threadCount, samples)

	// a channel to submit the test tasks
	testTasks := make(chan usize, threadCount)

	// a channel to receive results from the test tasks back on the main thread
	results := make(chan latency, samples)

	// create the workers for all the threads in this test
	for w := u1; w <= threadCount; w++ {
		go asyncObjectRequest(w, payloadSize, testTasks, results)
	}

	// start the timer for this benchmark
	statSample := statSample()
	benchmarkTimer := time.Now()

	// submit all the test tasks
	for j := u1; j <= samples; j++ {
		testTasks <- j
	}

	// close the channel
	close(testTasks)

	// construct a new benchmark record
	benchmarkRecord := benchmark{
		firstByte: make(map[stat]float64),
		lastByte:  make(map[stat]float64),
	}
	sumFirstByte := int64(0)
	sumLastByte := int64(0)
	benchmarkRecord.threads = int(threadCount)
	benchmarkRecord.sampleCount = samples

	// wait for all the results to come and collect the individual datapoints
	for s := u1; s <= samples; s++ {
		timing := <-results
		benchmarkRecord.dataPoints = append(benchmarkRecord.dataPoints, timing)
		sumFirstByte += timing.FirstByte.Nanoseconds()
		sumLastByte += timing.LastByte.Nanoseconds()
		benchmarkRecord.objectSize += payloadSize
	}

	// stop the timer for this benchmark
	totalTime := time.Now().Sub(benchmarkTimer)
	statAverage := statSample.averageToNow()

	// calculate the summary statistics for the first byte latencies
	sort.Sort(ByFirstByte(benchmarkRecord.dataPoints))
	benchmarkRecord.firstByte[avg] = (float64(sumFirstByte) / float64(samples)) / 1000000
	benchmarkRecord.firstByte[min] = float64(benchmarkRecord.dataPoints[0].FirstByte.Nanoseconds()) / 1000000
	benchmarkRecord.firstByte[max] = float64(benchmarkRecord.dataPoints[len(benchmarkRecord.dataPoints)-1].FirstByte.Nanoseconds()) / 1000000
	benchmarkRecord.firstByte[p25] = float64(benchmarkRecord.dataPoints[int(float64(samples)*float64(0.25))-1].FirstByte.Nanoseconds()) / 1000000
	benchmarkRecord.firstByte[p50] = float64(benchmarkRecord.dataPoints[int(float64(samples)*float64(0.5))-1].FirstByte.Nanoseconds()) / 1000000
	benchmarkRecord.firstByte[p75] = float64(benchmarkRecord.dataPoints[int(float64(samples)*float64(0.75))-1].FirstByte.Nanoseconds()) / 1000000
	benchmarkRecord.firstByte[p90] = float64(benchmarkRecord.dataPoints[int(float64(samples)*float64(0.90))-1].FirstByte.Nanoseconds()) / 1000000
	benchmarkRecord.firstByte[p99] = float64(benchmarkRecord.dataPoints[int(float64(samples)*float64(0.99))-1].FirstByte.Nanoseconds()) / 1000000

	// calculate the summary statistics for the last byte latencies
	sort.Sort(ByLastByte(benchmarkRecord.dataPoints))
	benchmarkRecord.lastByte[avg] = (float64(sumLastByte) / float64(samples)) / 1000000
	benchmarkRecord.lastByte[min] = float64(benchmarkRecord.dataPoints[0].LastByte.Nanoseconds()) / 1000000
	benchmarkRecord.lastByte[max] = float64(benchmarkRecord.dataPoints[len(benchmarkRecord.dataPoints)-1].LastByte.Nanoseconds()) / 1000000
	benchmarkRecord.lastByte[p25] = float64(benchmarkRecord.dataPoints[int(float64(samples)*float64(0.25))-1].LastByte.Nanoseconds()) / 1000000
	benchmarkRecord.lastByte[p50] = float64(benchmarkRecord.dataPoints[int(float64(samples)*float64(0.5))-1].LastByte.Nanoseconds()) / 1000000
	benchmarkRecord.lastByte[p75] = float64(benchmarkRecord.dataPoints[int(float64(samples)*float64(0.75))-1].LastByte.Nanoseconds()) / 1000000
	benchmarkRecord.lastByte[p90] = float64(benchmarkRecord.dataPoints[int(float64(samples)*float64(0.90))-1].LastByte.Nanoseconds()) / 1000000
	benchmarkRecord.lastByte[p99] = float64(benchmarkRecord.dataPoints[int(float64(samples)*float64(0.99))-1].LastByte.Nanoseconds()) / 1000000

	// calculate the throughput rate
	rate := (float64(benchmarkRecord.objectSize)) / (totalTime.Seconds()) / unitMB

	// determine what to put in the first column of the results
	c := benchmarkRecord.threads
	if throttlingMode {
		c = int(runNumber)
	}

	// print the results to stdout
	fmt.Printf("| %7d | \033[1;31m%9.1f MB/s\033[0m |%5.0f %5.0f %5.0f %5.0f %5.0f %5.0f %5.0f %5.0f |%5.0f %5.0f %5.0f %5.0f %5.0f %5.0f %5.0f %5.0f |\n",
		c, rate,
		benchmarkRecord.firstByte[avg], benchmarkRecord.firstByte[min], benchmarkRecord.firstByte[p25], benchmarkRecord.firstByte[p50], benchmarkRecord.firstByte[p75], benchmarkRecord.firstByte[p90], benchmarkRecord.firstByte[p99], benchmarkRecord.firstByte[max],
		benchmarkRecord.lastByte[avg], benchmarkRecord.lastByte[min], benchmarkRecord.lastByte[p25], benchmarkRecord.lastByte[p50], benchmarkRecord.lastByte[p75], benchmarkRecord.lastByte[p90], benchmarkRecord.lastByte[p99], benchmarkRecord.lastByte[max])

	commonLine := []string{
		fmt.Sprintf("%s", hostname),
		fmt.Sprintf("%s", instanceType),
		fmt.Sprintf("%d", payloadSize),
		fmt.Sprintf("%d", benchmarkRecord.threads),
	}

	// add the results to the csv array
	benchLine := append(commonLine, []string{
		fmt.Sprintf("%.3f", rate),
		fmt.Sprintf("%.1f", benchmarkRecord.firstByte[avg]),
		fmt.Sprintf("%.1f", benchmarkRecord.firstByte[min]),
		fmt.Sprintf("%.1f", benchmarkRecord.firstByte[p25]),
		fmt.Sprintf("%.1f", benchmarkRecord.firstByte[p50]),
		fmt.Sprintf("%.1f", benchmarkRecord.firstByte[p75]),
		fmt.Sprintf("%.1f", benchmarkRecord.firstByte[p90]),
		fmt.Sprintf("%.1f", benchmarkRecord.firstByte[p99]),
		fmt.Sprintf("%.1f", benchmarkRecord.firstByte[max]),
		fmt.Sprintf("%.2f", benchmarkRecord.lastByte[avg]),
		fmt.Sprintf("%.2f", benchmarkRecord.lastByte[min]),
		fmt.Sprintf("%.1f", benchmarkRecord.lastByte[p25]),
		fmt.Sprintf("%.1f", benchmarkRecord.lastByte[p50]),
		fmt.Sprintf("%.1f", benchmarkRecord.lastByte[p75]),
		fmt.Sprintf("%.1f", benchmarkRecord.lastByte[p90]),
		fmt.Sprintf("%.1f", benchmarkRecord.lastByte[p99]),
		fmt.Sprintf("%.1f", benchmarkRecord.lastByte[max]),
		fmt.Sprintf("%d", benchmarkRecord.sampleCount),
	}...)

	statLine := append(commonLine, statAverage.ToCsvRow()...)
	return benchLine, statLine
}

func asyncObjectRequest(o usize, payloadSize usize, tasks <-chan usize, results chan<- latency) {
	for range tasks {

		key := objectName // generateS3Key(hostname, o, payloadSize)
		byteRange := randomByteRange(objectSize, payloadSize)

		// start the timer to measure the first byte and last byte latencies
		latencyTimer := time.Now()

		// do the GetObject request
		req := s3Client.GetObjectRequest(&s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(key),
			Range:  aws.String(byteRange.ToHTTPHeader()),
		})

		resp, err := req.Send()

		// if a request fails, exit
		if err != nil {
			panic("Failed to get object: " + err.Error())
		}

		// measure the first byte latency
		firstByte := time.Now().Sub(latencyTimer)

		// create a buffer to copy the S3 object body to
		var buf = make([]byte, byteRange.Size())

		// read the s3 object body into the buffer
		size := 0
		for {
			n, err := resp.Body.Read(buf)

			size += n

			if err == io.EOF {
				break
			}

			// if the streaming fails, exit
			if err != nil {
				panic("Error reading object body: " + err.Error())
			}
		}

		_ = resp.Body.Close()

		// measure the last byte latency
		lastByte := time.Now().Sub(latencyTimer)

		// add the latency result to the results channel
		results <- latency{FirstByte: firstByte, LastByte: lastByte}
	}
}

// prints the table header for the test results
func printHeader(objectSize usize) {
	// instance type string used to render results to stdout
	instanceTypeString := ""

	if instanceType != "" {
		instanceTypeString = " (" + instanceType + ")"
	}

	// print the table header
	fmt.Printf("Download performance with \033[1;33m%-s\033[0m objects%s\n", byteFormat(float64(objectSize)), instanceTypeString)
	fmt.Println("                           +-------------------------------------------------------------------------------------------------+")
	fmt.Println("                           |            Time to First Byte (ms)             |            Time to Last Byte (ms)              |")
	fmt.Println("+---------+----------------+------------------------------------------------+------------------------------------------------+")
	if !throttlingMode {
		fmt.Println("| Threads |     Throughput |  avg   min   p25   p50   p75   p90   p99   max |  avg   min   p25   p50   p75   p90   p99   max |")
	} else {
		fmt.Println("|       # |     Throughput |  avg   min   p25   p50   p75   p90   p99   max |  avg   min   p25   p50   p75   p90   p99   max |")
	}
	fmt.Println("+---------+----------------+------------------------------------------------+------------------------------------------------+")
}

// cleans up the objects uploaded to S3 for this test (but doesn't remove the bucket)
func cleanup() {
	fmt.Print("\n---------- \033[1;32mCLEANUP\033[0m ----------\n\n")

	// fmt.Printf("Deleting any objects uploaded from %s\n", hostname)
	fmt.Printf("NO-OP")
	fmt.Print("\n\n")
}
