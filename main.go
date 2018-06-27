package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	argv := os.Args[1:len(os.Args)] // remove executable name
	exitFunction(goTestCheckCoverage(argv))
}

// injection point to enable test coverage
var exitFunction func(code int) = os.Exit

// Run go test with given arguments + coverage and inspect coverage after run
func goTestCheckCoverage(argv []string) (exitCode int) {
	// Run go test
	coveragePath := "coverage.out"
	exitCode = runGoTestWithCoverage(argv, coveragePath)
	if exitCode != 0 {
		return
	}
	exitCode = checkCoverage(coveragePath)
	return
}

func runGoTestWithCoverage(argv []string, coveragePath string) (exitCode int) {
	os.Remove(coveragePath)
	argv = append([]string{"test"}, argv...)
	argv = append(argv, "-cover", "-coverprofile="+coveragePath)
	return runCommand("go", argv...)
}

func checkCoverage(coveragePath string) (exitCode int) {
	// Tests passed, so let's check coverage for each file that has coverage
	uncoveredSections := uncoveredSections(coveragePath)
	pathSections := groupUncoveredSectionsByPath(uncoveredSections)
	iterateSorted(pathSections, func(path string, sections []string) {
		configured := configuredUncovered(path)
		current := len(sections)
		if current > configured {
			// TODO: color when tty
			fmt.Fprintf(os.Stderr, "%v new uncovered sections introduced (%v current vs %v configured)\n", path, current, configured)
			fmt.Fprintln(os.Stderr, strings.Join(sections, "\n"))
			exitCode = 1
		}
	})
	return
}

func groupUncoveredSectionsByPath(sections []string) (grouped map[string][]string) {
	grouped = map[string][]string{}
	for _, section := range sections {
		path := strings.Split(section, ":")[0]
		group, ok := grouped[path]
		if !ok {
			grouped[path] = []string{}
		}
		grouped[path] = append(group, section)
	}
	return
}

// Find the uncovered sections (file:line.char,line.char) given a coverage file
func uncoveredSections(coverageFilePath string) (sections []string) {
	content := readFile(coverageFilePath)

	sections = splitWithoutEmpty(content, '\n')
	if len(sections) == 0 {
		return []string{}
	}

	// remove the initial `set: mode` line
	sections = sections[1:]

	// find sections that are uncovered (end in " 0")
	sections = filter(sections, func(line string) bool { return strings.HasSuffix(line, " 0") })

	// remove coverage counters from sections
	sections = collect(sections, func(section string) string { return strings.Split(section, " ")[0] })

	// remove package prefix like "github.com/user/lib"
	wd, err := os.Getwd()
	check(err)
	sections = collect(sections, func(section string) string { return removeLocalPackageFromPath(section, wd) })

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
	content := readFile(path)
	regex := regexp.MustCompile("// *untested sections: *([0-9]+)")
	match := regex.FindStringSubmatch(content)
	if len(match) == 2 {
		coverted, err := strconv.Atoi(match[1])
		check(err)
		return coverted
	} else {
		return 0
	}
}
