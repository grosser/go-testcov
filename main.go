package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

func main() {
	argv := os.Args[1:len(os.Args)] // remove executable name
	exitFunction(goTestCheckCoverage(argv))
}

type Section struct {
	file      string
	startLine int
	startChar int
	endLine   int
	endChar   int
	sortvalue int
}

// covert raw coverage line into a section github.com/foo/bar/baz.go:1.2,3.5 1 0
func NewSection(raw string) Section {
	parts := strings.Split(raw, ":")
	file := parts[0]
	parts = strings.FieldsFunc(parts[1], func(r rune) bool { return r == '.' || r == ',' || r == ' ' })
	startLine := stringToInt(parts[0])
	startChar := stringToInt(parts[1])
	endLine := stringToInt(parts[2])
	endChar := stringToInt(parts[3])
	sortValue := startLine*100000 + startChar // we group by file, so we only need to sort by line+char
	return Section{file, startLine, startChar, endLine, endChar, sortValue}
}

func (s Section) Numbers() string {
	return fmt.Sprintf("%v.%v,%v.%v", s.startLine, s.startChar, s.endLine, s.endChar)
}

// injection point to enable test coverage
var exitFunction func(code int) = os.Exit

// Run go test with given arguments + coverage and inspect coverage after run
func goTestCheckCoverage(argv []string) (exitCode int) {
	// Run go test
	coveragePath := "coverage.out"
	os.Remove(coveragePath)
	defer os.Remove(coveragePath)

	exitCode = runGoTestWithCoverage(argv, coveragePath)
	if exitCode == 0 {
		exitCode = checkCoverage(coveragePath)
	}

	return
}

func runGoTestWithCoverage(argv []string, coveragePath string) (exitCode int) {
	argv = append([]string{"test"}, argv...)
	argv = append(argv, "-coverprofile", coveragePath)
	return runCommand("go", argv...)
}

func checkCoverage(coveragePath string) (exitCode int) {
	// Tests passed, so let's check coverage for each file that has coverage
	uncoveredSections := uncoveredSections(coveragePath)
	pathSections := groupSectionsByPath(uncoveredSections)
	wd, err := os.Getwd()
	check(err)

	iterateSorted(pathSections, func(path string, sections []Section) {
		configured := configuredUncovered(path)
		current := len(sections)
		if current > configured {
			// remove package prefix like "github.com/user/lib", but cache the call to os.Getwd
			path = removeLocalPackageFromPath(path, wd)

			// TODO: color when tty
			fmt.Fprintf(os.Stderr, "%v new uncovered sections introduced (%v current vs %v configured)\n", path, current, configured)

			// sort sections since go does not
			sort.Slice(sections, func(i, j int) bool {
				return sections[i].sortvalue < sections[j].sortvalue
			})

			for _, section := range sections {
				// copy-paste friendly snippets
				fmt.Fprintln(os.Stderr, path+":"+section.Numbers())
			}

			exitCode = 1
		}
	})
	return
}

func groupSectionsByPath(sections []Section) (grouped map[string][]Section) {
	grouped = map[string][]Section{}
	for _, section := range sections {
		path := section.file
		group, ok := grouped[path]
		if !ok {
			grouped[path] = []Section{}
		}
		grouped[path] = append(group, section)
	}
	return
}

// Find the uncovered sections (file:line.char,line.char) given a coverage file
func uncoveredSections(coverageFilePath string) (sections []Section) {
	sections = []Section{}
	content := readFile(coverageFilePath)

	lines := splitWithoutEmpty(content, '\n')

	// remove the initial `set: mode` line
	if len(lines) == 0 {
		return
	}
	lines = lines[1:]

	// we want lines that end in " 0", they have no coverage
	for _, line := range lines {
		if strings.HasSuffix(line, " 0") {
			sections = append(sections, NewSection(line))
		}
	}

	return
}

// turn foo.com/foo/bar/a.go into a.go if we are in a directory that ends with foo.com/foo/bar
func removeLocalPackageFromPath(path string, workingDirectory string) string {
	prefixSize := 3
	separator := string(os.PathSeparator)
	parts := strings.Split(path, separator)
	if len(parts) <= prefixSize {
		return path
	}

	prefix := strings.Join(parts[:prefixSize], separator)
	if strings.HasSuffix(workingDirectory, prefix) {
		return strings.Split(path, prefix+separator)[1]
	}

	return path
}

// How many sections are expected to be uncovered, 0 if not configured
func configuredUncovered(path string) (count int) {
	content := readFile(joinPath(os.Getenv("GOPATH"), "src", path))
	regex := regexp.MustCompile("// *untested sections: *([0-9]+)")
	match := regex.FindStringSubmatch(content)
	if len(match) == 2 {
		coverted := stringToInt(match[1])
		return coverted
	} else {
		return 0
	}
}
