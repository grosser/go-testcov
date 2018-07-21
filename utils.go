package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

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

func joinPath(parts ...string) string {
	return strings.Join(parts, string(os.PathSeparator))
}

func stringToInt(string string) int {
	coverted, err := strconv.Atoi(string)
	check(err)
	return coverted
}
