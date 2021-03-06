//
// This is the webhook bridge, which should be built like so:
//
//     go build .
//
// Once built launch it as follows:
//
//     $ ./webhook-bridge -url=https://example.com/bla
//
// When a test fails a webhook will sent
//
// Alberto
// --
//

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/cmaster11/overseer/test"
	"github.com/go-redis/redis"
)

// The url we notify
var webhookURL *string
var sendTestSuccess *bool
var sendTestRecovered *bool

// The redis handle
var r *redis.Client

//
// Given a JSON string decode it and post it via webhook if it describes
// a test-failure.
//
func process(msg []byte) {
	testResult, err := test.ResultFromJSON(msg)
	if err != nil {
		panic(err)
	}

	// If the test passed then we don't care, unless otherwise defined
	shouldSend := true
	if testResult.Error == nil {
		shouldSend = false

		if *sendTestSuccess {
			shouldSend = true
		}

		if *sendTestRecovered && testResult.Recovered {
			shouldSend = true
		}
	}

	if !shouldSend {
		return
	}

	fmt.Printf("Processing result: %+v\n", testResult)

	res, err := http.Post(*webhookURL, "application/json", bytes.NewBuffer(msg))
	if err != nil {
		fmt.Printf("Failed to execute webhook request: %s\n", err.Error())
		return
	}

	//
	// OK now we've submitted the post.
	//
	// We should retrieve the status-code + body, if the status-code
	// is "odd" then we'll show them.
	//
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("Error reading response to post: %s\n", err.Error())
		return
	}
	status := res.StatusCode

	if status < 200 || status >= 400 {
		fmt.Printf("Error - Status code was not successful: %d\n", status)
		fmt.Printf("Response - %s\n", body)
	}
}

//
// Entry Point
//
func main() {

	//
	// Parse our flags
	//
	redisHost := flag.String("redis-host", "127.0.0.1:6379", "Specify the address of the redis queue.")
	redisPass := flag.String("redis-pass", "", "Specify the password of the redis queue.")
	redisQueueKey := flag.String("redis-queue-key", "overseer.results", "Specify the redis queue key to use.")

	webhookURL = flag.String("url", "", "The url address to notify")
	sendTestSuccess = flag.Bool("send-test-success", false, "Send also test results when successful")
	sendTestRecovered = flag.Bool("send-test-recovered", false, "Send also test results when a test recovers from failure (valid only when used together with deduplication rules)")
	flag.Parse()

	//
	// Sanity-check.
	//
	if *webhookURL == "" {
		fmt.Printf("Usage: webhook-bridge -url=https://example.com/bla [-redis-host=127.0.0.1:6379] [-redis-pass=foo]\n")
		os.Exit(1)
	}

	_, err := url.Parse(*webhookURL)
	if err != nil {
		fmt.Printf("Failed to parse provided URL: %s\n", err.Error())
		os.Exit(1)
	}

	//
	// Create the redis client
	//
	r = redis.NewClient(&redis.Options{
		Addr:     *redisHost,
		Password: *redisPass,
		DB:       0, // use default DB
	})

	//
	// And run a ping, just to make sure it worked.
	//
	_, err = r.Ping().Result()
	if err != nil {
		fmt.Printf("Redis connection failed: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("webhook bridge started with url %s\n", *webhookURL)

	for {

		//
		// Get test-results
		//
		msg, _ := r.BLPop(0, *redisQueueKey).Result()

		//
		// If they were non-empty, process them.
		//
		//   msg[0] will be "overseer.results"
		//
		//   msg[1] will be the value removed from the list.
		//
		if len(msg) >= 1 {
			process([]byte(msg[1]))
		}
	}
}
