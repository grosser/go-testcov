package main

import (
	"io/ioutil"
	"strings"
	"os/exec"
	"syscall"
	"os"
	"fmt"
	"regexp"
	"strconv"
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

// Run go test with given arguments + coverage and inspect coverage after run
func covTest(argv []string) (exitCode int) {
	coveragePath := "coverage.out"
	os.Remove(coveragePath)
	argv = append([]string{"test"}, argv...)
	argv = append(argv, "-cover", "-coverprofile=" +coveragePath)
	exitCode = runCommand("go", argv...)
	if(exitCode != 0) {
		return
	}

	// Tests passed, so let's check coverage
	uncovered := uncovered(coveragePath)
	if(len(uncovered) == 0) {
		return // NOTE: Ideally warn when coverage is below configured, but then we don't know what file was covered
	}

	// TODO: support multiple files
	fileUnderTest := strings.Split(uncovered[0], ":")[0]
	configuredUncovered := expectedUncovered(fileUnderTest)
	actualUncovered := len(uncovered)

	if(configuredUncovered < actualUncovered) {
		// TODO: color
		fmt.Fprintf(os.Stderr, "%v uncovered sections found, but expected %v:\n", actualUncovered, configuredUncovered)
		fmt.Fprintln(os.Stderr, strings.Join(uncovered, "\n"))
		exitCode = 1
	}
	return
}

// Find the uncovered lines given a coverage file
func uncovered(path string) (uncoveredLines []string) {
	content := readFile(path)

	lines := splitWithoutEmpty(content, '\n')
	if (len(lines) == 0) {
		return []string{}
	}

	// remove the initial `set: mode` line
	lines = lines[1:]

	// find lines that are uncovered (end in " 0")
	lines = filter(lines, func(line string) bool { return strings.HasSuffix(line, " 0") })

	// remove converage info from lines
	lines = collect(lines, func(line string) string { return strings.Split(line, " ")[0] })

	return lines
}

// How many sections are expected to be uncovered, 0 if not configured
func expectedUncovered(path string) (count int) {
	content := readFile(path)
	regex := regexp.MustCompile("// *untested sections: *([0-9]+)")
	match := regex.FindStringSubmatch(content)
	if(len(match) == 2) {
		coverted, err := strconv.Atoi(match[1])
		check(err)
		return coverted
	} else {
		return 0
	}
}

func main(){
	argv := os.Args[1:len(os.Args)] // remove executable name
	exitFunction(covTest(argv))
}