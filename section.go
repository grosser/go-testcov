package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Section represents a line as produced by `go test`
type Section struct {
	pkg       string
	startLine int
	startChar int
	endLine   int
	endChar   int
	sortValue int
}

// NewSection parses a coverage line as produces by `go test`, for example "github.com/foo/bar/baz.go:1.2,3.5 1 0"
func NewSection(line string) Section {
	// parse which package was covered
	fileAndLocation := strings.SplitN(line, ":", 2)
	pkg := fileAndLocation[0]
	location := fileAndLocation[1]

	// parse where the coverage starts and ends
	locations := regexp.MustCompile("[,. ]").Split(location, -1)
	startLine := stringToInt(locations[0])
	startChar := stringToInt(locations[1])
	endLine := stringToInt(locations[2])
	endChar := stringToInt(locations[3])

	// allow sorting multiple sections from the same pkg
	sortValue := startLine*100000 + startChar

	return Section{pkg, startLine, startChar, endLine, endChar, sortValue}
}

func (s Section) Location() string {
	return fmt.Sprintf("%v.%v,%v.%v", s.startLine, s.startChar, s.endLine, s.endChar)
}
