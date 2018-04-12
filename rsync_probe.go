package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

//
// Our structure.
//
// We store state in the `input` field.
//
type RSYNCTest struct {
	input string
}

//
// Run the test against the specified target.
//
func (s *RSYNCTest) runTest(target string) error {
	var err error

	//
	// The default port to connect to.
	//
	port := 873

	//
	// If the user specified a different port update it.
	//
	re := regexp.MustCompile("on\\s+port\\s+([0-9]+)")
	out := re.FindStringSubmatch(s.input)
	if len(out) == 2 {
		port, err = strconv.Atoi(out[1])
		if err != nil {
			return err
		}
	}

	//
	// Set an explicit timeout
	//
	d := net.Dialer{Timeout: TIMEOUT}

	//
	// Default to connecting to an IPv4-address
	//
	address := fmt.Sprintf("%s:%d", target, port)

	//
	// If we find a ":" we know it is an IPv6 address though
	//
	if strings.Contains(target, ":") {
		address = fmt.Sprintf("[%s]:%d", target, port)
	}

	//
	// Make the TCP connection.
	//
	conn, err := d.Dial("tcp", address)
	if err != nil {
		return err
	}

	//
	// Read the banner.
	//
	banner, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return err
	}
	conn.Close()

	if !strings.Contains(banner, "RSYNC") {
		return errors.New("Banner doesn't look like an rsync-banner")
	}

	return nil
}

//
// Store the complete line from the parser in our private
// field; this could be used if there are protocol-specific options
// to be understood.
//
func (s *RSYNCTest) setLine(input string) {
	s.input = input
}

//
// Register our protocol-tester.
//
func init() {
	Register("rsync", func() ProtocolTest {
		return &RSYNCTest{}
	})
}
