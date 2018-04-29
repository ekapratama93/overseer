// Local
//
// The local sub-command executes parsed tests locally.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/google/subcommands"
	"github.com/skx/overseer/parser"
	"github.com/skx/overseer/test"
)

type localCmd struct {
	IPv4    bool
	IPv6    bool
	MQ      string
	Retry   bool
	Timeout int
	Verbose bool
}

//
// Glue
//
func (*localCmd) Name() string     { return "local" }
func (*localCmd) Synopsis() string { return "Execute tests locally" }
func (*localCmd) Usage() string {
	return `local :
  Execute the tests in the given files locally, without the use of a queue.
`
}

//
// Flag setup.
//
func (p *localCmd) SetFlags(f *flag.FlagSet) {

	//
	// Create the default options here
	//
	// This is done so we can load defaults via a configuration-file
	// if present.
	//
	var defaults localCmd
	defaults.IPv4 = true
	defaults.IPv6 = true
	defaults.MQ = ""
	defaults.Retry = true
	defaults.Timeout = 10
	defaults.Verbose = false

	//
	// If we have a configuration file then load it
	//
	if len(os.Getenv("OVERSEER")) > 0 {
		cfg, err := ioutil.ReadFile(os.Getenv("OVERSEER"))
		if err == nil {
			err = json.Unmarshal(cfg, &defaults)
			if err != nil {
				fmt.Printf("WARNING: Error loading overseer.json - %s\n",
					err.Error())
			}
		} else {
			fmt.Printf("WARNING: Failed to read configuration-file - %s\n",
				err.Error())
		}
	}

	//
	// Otherwise set the defaults and flags.
	//
	f.BoolVar(&p.Verbose, "verbose", defaults.Verbose, "Show more output.")
	f.BoolVar(&p.IPv4, "4", defaults.IPv4, "Enable IPv4 tests.")
	f.BoolVar(&p.IPv6, "6", defaults.IPv6, "Enable IPv6 tests.")
	f.BoolVar(&p.Retry, "retry", defaults.Retry, "Should failing tests be retried a few times before raising a notification.")
	f.IntVar(&p.Timeout, "timeout", defaults.Timeout, "The global timeout for all tests, in seconds.")
	f.StringVar(&p.MQ, "mq", defaults.MQ, "Specify the MQ address to connect to.")
}

//
// This is a callback invoked by the parser when a job
// has been successfully parsed.
//
// Run the test, via our helper
//
func (p *localCmd) runTest(tst test.Test) error {

	//
	// Setup the options for the test.
	//
	var opts test.TestOptions
	opts.Verbose = p.Verbose
	opts.IPv4 = p.IPv4
	opts.IPv6 = p.IPv6
	opts.Retry = p.Retry
	opts.Timeout = time.Duration(p.Timeout) * time.Second

	//
	// Now run the test.
	//
	return runTest(tst, opts)
}

//
// Entry-point.
//
func (p *localCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	//
	// If the MQ address is configured the connect.
	//
	if p.MQ != "" {
		err := ConnectMQ(p.MQ)

		if err != nil {
			fmt.Printf("Failed to connect to MQ: %s\n", err.Error())
			os.Exit(1)
		}
	}

	//
	// We'll run tests from each file passed as an argument.
	//
	for _, file := range f.Args() {

		//
		// Create an object to parse our file.
		//
		helper := parser.New()

		//
		// Invoke the run_test callback to execute each test.
		//
		err := helper.ParseFile(file, p.runTest)
		if err != nil {
			fmt.Printf("Error parsing file: %s\n", err.Error())
		}
	}

	return subcommands.ExitSuccess
}
