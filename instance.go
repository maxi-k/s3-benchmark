package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const instanceDataUrlPrefix = "http://169.254.169.254/latest/meta-data"

func instanceDataUrl(suffix string) string {
	return fmt.Sprintf("%s/%s", instanceDataUrlPrefix, suffix)
}

// gets the hostname or the EC2 instance ID
func getHostname() string {
	instanceId := getInstanceId()
	if instanceId != "" {
		return instanceId
	}

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return hostname
}

// gets the EC2 region from the instance metadata
func getRegion() string {
	httpClient := &http.Client{
		Timeout: time.Second,
	}

	link := instanceDataUrl("placement/availability-zone")
	response, err := httpClient.Get(link)
	if err != nil {
		return defaultRegion
	}

	content, _ := ioutil.ReadAll(response.Body)
	_ = response.Body.Close()

	az := string(content)

	return az[:len(az)-1]
}

// gets the EC2 instance type from the instance metadata
func getInstanceType() string {
	httpClient := &http.Client{
		Timeout: time.Second,
	}

	link := instanceDataUrl("instance-type")
	response, err := httpClient.Get(link)
	if err != nil {
		return "unknown-instance"
	}

	content, _ := ioutil.ReadAll(response.Body)
	_ = response.Body.Close()

	return string(content)
}

// gets the EC2 instance ID from the instance metadata
func getInstanceId() string {
	httpClient := &http.Client{
		Timeout: time.Second,
	}

	link := instanceDataUrl("meta-data/instance-id")
	response, err := httpClient.Get(link)
	if err != nil {
		return ""
	}

	content, _ := ioutil.ReadAll(response.Body)
	_ = response.Body.Close()

	return string(content)
}

// generates an S3 key from the sha hash of the hostname, thread index, and object size
func generateS3Key(host string, threadIndex int, payloadSize usize) string {
	keyHash := sha1.Sum([]byte(fmt.Sprintf("%s-%03d-%012d", host, threadIndex, payloadSize)))
	key := fmt.Sprintf("%x", keyHash)
	return key
}

func uploadCsv(prefix string, data [][]string) {
	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	_ = w.WriteAll(data)

	key := prefix + ".csv"

	// do the PutObject request
	putReq := s3Client.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    &key,
		Body:   bytes.NewReader(b.Bytes()),
	})

	_, err := putReq.Send()

	// if the request fails, exit
	if err != nil {
		panic("Failed to put object: " + err.Error())
	}

	fmt.Printf("CSV data uploaded to \033[1;33ms3://%s/%s\033[0m\n", bucketName, key)
}
