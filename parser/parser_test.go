//
//  Basic testing of our DB primitives
//

package parser

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

// Test that parsing a missing file returns an error
func TestMissingFile(t *testing.T) {

	p := New()
	err := p.ParseFile("/path/is/not/found", nil)

	if err == nil {
		t.Errorf("Parsing a missing file didn't raise an error!")
	}
}

// Test reading samples from a file
func TestFile(t *testing.T) {
	file, err := ioutil.TempFile(os.TempDir(), "prefix")
	defer os.Remove(file.Name())

	// Write to the file
	lines := `
http://example.com/ must run http
# This is fine
http://example.com/ must run http with content 'moi'

# The content-type here will not match
http://example.com/ must run http with content "moi"
`
	//
	err = ioutil.WriteFile(file.Name(), []byte(lines), 0644)
	if err != nil {
		t.Errorf("Error writing our test-case")
	}

	//
	// Now parse the file
	//
	p := New()
	err = p.ParseFile(file.Name(), nil)

	if err != nil {
		t.Errorf("Error parsing our valid file")
	}
}

// Test reading macro-based samples from a file
func TestFileMacro(t *testing.T) {
	file, err := ioutil.TempFile(os.TempDir(), "prefix")
	defer os.Remove(file.Name())

	// Write to the file
	lines := `
FOO are host1.example.com, host2.example.com
FOO must run ssh
`
	//
	err = ioutil.WriteFile(file.Name(), []byte(lines), 0644)
	if err != nil {
		t.Errorf("Error writing our test-case")
	}

	//
	// Now parse the file
	//
	p := New()
	err = p.ParseFile(file.Name(), nil)

	if err != nil {
		t.Errorf("Error parsing our valid file")
	}
}

// Test redefinining macros is a bug.
func TestFileMacroRedefined(t *testing.T) {
	file, err := ioutil.TempFile(os.TempDir(), "prefix")
	defer os.Remove(file.Name())

	// Write to the file
	lines := `
FOO are host1.example.com, host2.example.com
FOO must run ssh
FOO are host3.example.com, host4.example.com
FOO must run ftp
`
	//
	err = ioutil.WriteFile(file.Name(), []byte(lines), 0644)
	if err != nil {
		t.Errorf("Error writing our test-case")
	}

	//
	// Now parse the file
	//
	p := New()
	err = p.ParseFile(file.Name(), nil)

	if err == nil {
		t.Errorf("Expected error parsing file, didn't see one!")
	}
	if !strings.Contains(err.Error(), "Redeclaring an existing macro") {
		t.Errorf("The expected error differed from what we received")
	}
}

// Test some valid input
func TestValidLines(t *testing.T) {

	var inputs = []string{
		"foo must run http",
		"bar must run http",
		"baz must run ftp"}

	for _, line := range inputs {

		p := New()
		_, err := p.ParseLine(line, nil)

		if err != nil {
			t.Errorf("Found error parsing valid line: %s\n", err.Error())
		}
	}
}

// Test some malformed lines
func TestUnknownInput(t *testing.T) {

	var inputs = []string{
		"foo must RAN blah",
		"bar mustn't exist",
		"baz must ping"}

	for _, line := range inputs {

		p := New()
		_, err := p.ParseLine(line, nil)

		if err == nil {
			t.Errorf("Should have found error parsing line: %s\n", err.Error())
		}
		if !strings.Contains(err.Error(), "Unrecognized line") {
			t.Errorf("Received unexpected error: %s\n", err.Error())
		}
	}
}

// Test some invalid inputs
func TestUnknownProtocols(t *testing.T) {

	var inputs = []string{
		"foo must run blah",
		"bar must run moi",
		"baz must run kiss"}

	for _, line := range inputs {

		p := New()
		_, err := p.ParseLine(line, nil)

		if err == nil {
			t.Errorf("Should have found error parsing line: %s\n", err.Error())
		}
		if !strings.Contains(err.Error(), "Unknown test-type") {
			t.Errorf("Received unexpected error: %s\n", err.Error())
		}
	}
}

// Test parsing things that should return no options
func TestNoArguments(t *testing.T) {

	tests := []string{
		"127.0.0.1 must run ping",
		"127.0.0.1 must run ssh",
	}

	// Create a parser
	p := New()

	// Parse each line
	for _, input := range tests {

		out, err := p.ParseLine(input, nil)
		if err != nil {
			t.Errorf("Error parsing %s - %s", input, err.Error())
		}
		if len(out.Arguments) != 0 {
			t.Errorf("Surprising output")
		}
	}
}

// Test parsing some common HTTP options
func TestHTTPOptions(t *testing.T) {

	tests := []string{
		"http://example.com/ must run http with content 'moi' and ..",
		"http://example.com/ must run http with content moi",
		"http://example.com/ must run http with status '200'",
		"http://example.com/ must run http with status 200",
	}

	// Create a parser
	p := New()

	// Parse each line
	for _, input := range tests {

		// Parse the line
		out, err := p.ParseLine(input, nil)
		if err != nil {
			t.Errorf("Error parsing %s - %s", input, err.Error())
		}

		// We should have a single argument in each case
		if len(out.Arguments) != 1 {
			t.Errorf("Surprising output - we expected 1 option but found %d", len(out.Arguments))
		}
	}
}

// Test quotation-removal
func TestQuoteRemoval(t *testing.T) {

	tests := []string{
		"http://example.com/ must run http with content 'moi' and ..",
		"http://example.com/ must run http with content \"moi\"",
		"http://example.com/ must run http with content moi",
	}

	// Create a parser
	p := New()

	// Parse each line
	for _, input := range tests {

		out, err := p.ParseLine(input, nil)
		if err != nil {
			t.Errorf("Error parsing %s - %s", input, err.Error())
		}

		// We expect one parameter: content
		if len(out.Arguments) != 1 {
			t.Errorf("Surprising output - we expected 1 option but found %d", len(out.Arguments))
		}

		// The value should be 'moi'
		if out.Arguments["content"] != "moi" {
			t.Errorf("We expected the key 'content' to have the value 'moi', but found %s", out.Arguments["content"])
		}
	}
}

// Test quotation-removal doesn't modify the content of a string
func TestQuoteRemovalSanity(t *testing.T) {

	tests := []string{
		"http://example.com/ must run http with content 'm\"'oi' and ..",
		"http://example.com/ must run http with content \"m\"'oi\"",
		"http://example.com/ must run http with content m\"'oi",
	}

	// Create a parser
	p := New()

	// Parse each line
	for _, input := range tests {

		out, err := p.ParseLine(input, nil)
		if err != nil {
			t.Errorf("Error parsing %s - %s", input, err.Error())
		}

		// We expect one parameter: content
		if len(out.Arguments) != 1 {
			t.Errorf("Surprising output - we expected 1 option but found %d", len(out.Arguments))
		}

		// The value should have a single quote and double-quote
		single := 0
		double := 0
		for _, c := range out.Arguments["content"] {
			if c == '"' {
				double += 1
			}
			if c == '\'' {
				single += 1
			}
		}

		if single != 1 {
			t.Errorf("We found the wrong number of single-quotes: %d != 1", single)

		}
		if double != 1 {
			t.Errorf("We found the wrong number of double-quotes: %d != 1", double)
		}
	}
}

// Test a real line
func TestReal(t *testing.T) {
	in := "http://steve.fi/ must run http with status 301 with content 'Steve Kemp'"

	// Create a parser
	p := New()

	out, err := p.ParseLine(in, nil)
	if err != nil {
		t.Errorf("Error parsing %s - %s", in, err.Error())
	}

	// We expect two parameter: content + status
	if len(out.Arguments) != 2 {
		t.Errorf("Received the wrong number of parameters")
	}
	if out.Arguments["status"] != "301" {
		t.Errorf("Failed to get the correct status-value")
	}
	if out.Arguments["content"] != "Steve Kemp" {
		t.Errorf("Failed to get the correct content-value")
	}

}
