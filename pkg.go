package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// injection point to enable test coverage
var exitFunction func(code int) = os.Exit

// Util: blow up on errors without extra conditionals everywhere
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Util: "" => []  "foo" => ["foo"]
func splitWithoutEmpty(string string, delimiter rune) []string {
	return strings.FieldsFunc(string, func(c rune) bool { return c == delimiter })
}

// Util: select only the items from an array that match the given function
func filter(ss []string, test func(string) bool) (filtered []string) {
	filtered = []string{}
	for _, s := range ss {
		if test(s) {
			filtered = append(filtered, s)
		}
	}
	return
}

// Util: map each item of an array to what a function returns
func collect(vs []string, f func(string) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

// Util:
// Run a command and stream output to stdout/err, but return an exit code
// https://stackoverflow.com/questions/10385551/get-exit-code-go
func runCommand(name string, args ...string) (exitCode int) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			// This will happen (in OSX) if `name` is not available in $PATH,
			// in this situation, exit code could not be get
			fmt.Fprintf(os.Stderr, "Could not get exit code for failed program: %v, %v\n", name, args)
			exitCode = 1
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}
	return
}

// Util: read a file into a string
func readFile(path string) (content string) {
	data, err := ioutil.ReadFile(path)
	check(err)
	return string(data)
}

// Util: iterate a map in sorted way
func iterateSorted(data map[string][]string, fn func(string, []string)) {
	keys := make([]string, len(data))
	i := 0
	for k := range data {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		fn(k, data[k])
	}
}

// Run go test with given arguments + coverage and inspect coverage after run
func covTest(argv []string) (exitCode int) {
	// Run go test
	coveragePath := "coverage.out"
	exitCode = runGoTestWithCoverage(argv, coveragePath)
	if exitCode != 0 {
		return
	}

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

// NOTE: more efficient to return a nested+sorted array, could also keep track of current path and switch keys
// when the path changes since I'd expect go to dump coverage sorted by path
// but be careful not to sort sections since that would sort foo:10 before foo:2
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

func runGoTestWithCoverage(argv []string, coveragePath string) (exitCode int) {
	os.Remove(coveragePath)
	argv = append([]string{"test"}, argv...)
	argv = append(argv, "-cover", "-coverprofile="+coveragePath)
	return runCommand("go", argv...)
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

	// remove package prefix like "github.com/user/lib" from filename if that is the current directory
	wd, err := os.Getwd()
	check(err)
	prefixSize := 3 // github.com/foo/bar
	sections = collect(sections, func(section string) string {
		separator := string(os.PathSeparator)
		parts := strings.Split(section, separator)
		if len(parts) <= prefixSize {
			return section
		}

		prefix := strings.Join(parts[:prefixSize], separator)
		if strings.HasSuffix(wd, prefix) {
			return strings.Split(section, prefix+separator)[1]
		}

		return section

	})

	return
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

func main() {
	argv := os.Args[1:len(os.Args)] // remove executable name
	exitFunction(covTest(argv))
}
