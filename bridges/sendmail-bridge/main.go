//
// This is the sendmail bridge, which should be built like so:
//
//     go build .
//
// Once built launch it as follows:
//
//     $ ./sendmail-bridge -email=sysadmin@example.com
//
// When a test fails an email will sent via sendmail
//
// Steve
// --
//

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"text/template"

	"github.com/cmaster11/overseer/test"

	"github.com/go-redis/redis"
)

// The redis handle
var r *redis.Client

// Template is our text/template which is used to generate the email
// notification to the user.
var Template = `From: {{.From}}
To: {{.To}}
Subject: The {{.Type}} test failed against {{.Target}}

The {{.Type}} test failed against {{.Target}}.

The complete test was:

   {{.Input}}

The failure was:

   {{.Failure}}

`

type EmailBridge struct {
	// The email we notify
	Email string
}

//
// Given a JSON string decode it and post it via email if it describes
// a test-failure.
//
func (bridge *EmailBridge) Process(msg []byte) {
	testResult, err := test.ResultFromJSON(msg)
	if err != nil {
		panic(err)
	}

	//
	// If the test passed then we don't care.
	//
	if testResult.Error == nil {
		return
	}

	//
	// Here is a temporary structure we'll use to popular our email
	// template.
	//
	type TemplateParms struct {
		To      string
		From    string
		Target  string
		Type    string
		Input   string
		Failure string
	}

	//
	// Populate it appropriately.
	//
	var x TemplateParms
	x.To = bridge.Email
	x.From = bridge.Email
	x.Type = testResult.Type
	x.Target = testResult.Target
	x.Input = testResult.Input
	x.Failure = *testResult.Error

	//
	// Render our template into a buffer.
	//
	src := string(Template)
	t := template.Must(template.New("tmpl").Parse(src))
	buf := &bytes.Buffer{}
	err = t.Execute(buf, x)
	if err != nil {
		fmt.Printf("Failed to compile email-template %s\n", err.Error())
		return
	}

	//
	// Prepare to run sendmail, with a pipe we can write our message to.
	//
	sendmail := exec.Command("/usr/sbin/sendmail", "-f", bridge.Email, bridge.Email)
	stdin, err := sendmail.StdinPipe()
	if err != nil {
		fmt.Printf("Error sending email: %s\n", err.Error())
		return
	}

	//
	// Get the output pipe.
	//
	stdout, err := sendmail.StdoutPipe()
	if err != nil {
		fmt.Printf("Error sending email: %s\n", err.Error())
		return
	}

	//
	// Run the command, and pipe in the rendered template-result
	//
	sendmail.Start()
	_, err = stdin.Write(buf.Bytes())
	if err != nil {
		fmt.Printf("Failed to write to sendmail pipe: %s\n", err.Error())
	}
	stdin.Close()

	//
	// Read the output of Sendmail.
	//
	_, err = ioutil.ReadAll(stdout)
	if err != nil {
		fmt.Printf("Error reading mail output: %s\n", err.Error())
		return
	}

	err = sendmail.Wait()

	if err != nil {
		fmt.Printf("Waiting for process to terminate failed: %s\n", err.Error())
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
	var email = flag.String("email", "", "The email address to notify")
	flag.Parse()

	//
	// Sanity-check.
	//
	if *email == "" {
		fmt.Printf("Usage: email-bridge -email=sysadmin@example.com [-redis-host=127.0.0.1:6379] [-redis-pass=foo]\n")
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
	_, err := r.Ping().Result()
	if err != nil {
		fmt.Printf("Redis connection failed: %s\n", err.Error())
		os.Exit(1)
	}

	bridge := EmailBridge{
		Email: *email,
	}

	for {

		//
		// Get test-results
		//
		msg, _ := r.BLPop(0, "overseer.results").Result()

		//
		// If they were non-empty, process them.
		//
		//   msg[0] will be "overseer.results"
		//
		//   msg[1] will be the value removed from the list.
		//
		if len(msg) >= 1 {
			bridge.Process([]byte(msg[1]))
		}
	}
}
