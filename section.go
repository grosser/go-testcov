package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Section represents a line as produced by `go test`
type Section struct {
	path      string
	startLine int
	startChar int
	endLine   int
	endChar   int
	sortValue int
	callCount int
}

// NewSection parses a coverage line as produces by `go test`, for example "foo/bar.go:1.2,3.5 1 0"
func NewSection(line string) Section {
	// parse which package was covered
	fileAndLocation := strings.SplitN(line, ":", 2)
	path := fileAndLocation[0]
	location := fileAndLocation[1]

	// parse where the coverage starts and ends
	locations := regexp.MustCompile("[,. ]").Split(location, -1)
	startLine := stringToInt(locations[0])
	startChar := stringToInt(locations[1])
	endLine := stringToInt(locations[2])
	endChar := stringToInt(locations[3])

	// allow sorting multiple sections from the same path
	sortValue := startLine*100000 + startChar

	callCount := stringToInt(locations[len(locations)-1])

	return Section{path, startLine, startChar, endLine, endChar, sortValue, callCount}
}

func (s Section) Location() string {
	return fmt.Sprintf("%v.%v,%v.%v", s.startLine, s.startChar, s.endLine, s.endChar)
}
