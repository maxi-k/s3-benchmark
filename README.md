# S3 Benchmark

This project is forked from [David
Vassallo](https://github.com/dvassallo/s3-benchmark), and fits it to
my specific use-case.

**Notable Changes:**
- Measure CPU usage and other metrics while executing the
  benchmark as well, upload the results to S3
- Use one very large S3 object where different byte-ranges are fetched
  from, instead of multiple smaller objects. It needs to be pre-created.
- Increase sizes of fetched chunks from a few KB to multiple MB
- Calculate the tested thread counts from the number of cores /
  hardware threads on the machine
- Provide script which generates EC2 instance roles with minimal
  required permissions for the benchmark


### Run

Make the file executable:

```
chmod +x s3-benchmark
```

Run a quick test (takes a few minutes):
```
./s3-benchmark
```

Or run the full test (takes a few hours):
```
./s3-benchmark -full
```

See [this](https://github.com/dvassallo/s3-benchmark/blob/master/main.go#L123-L134) for all the other options.

### Build

1. Install [Go](https://golang.org/)
    ```
    sudo apt-get install golang-go
    ```
    or
    ```
    sudo yum install go
    ```
    may work too.

2. Setup Go environment variables (Usually GOPATH and GOBIN) and test Go installation
3. Clone the repo
4. Run ```go run *.go``` in this directory


## License

This project is forked from [David
Vassallo](https://github.com/dvassallo/s3-benchmark), and thus also
released under the [MIT License](LICENSE).
