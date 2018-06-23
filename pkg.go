package go_scov

import (
	"io/ioutil"
	"strings"
	"os/exec"
	"syscall"
	"os"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// "" => []  "foo" => ["foo"]
func splitWithoutEmpty(string string, delimiter rune) []string {
	return strings.FieldsFunc(string, func(c rune) bool { return c == delimiter })
}

func filter(ss []string, test func(string) bool) []string {
	ret := []string{}
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return ret
}

// TODO move into a utils package ? ... how does coverage work then ?
// Run a command and stream output to stdout/err, but return an exit code
// https://stackoverflow.com/questions/10385551/get-exit-code-go
const defaultFailedCode = 1

func runCommand(name string, argv []string) (exitCode int) {
	cmd := exec.Command(name, argv...)
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
			exitCode = defaultFailedCode
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}
	return
}

// TODO use multi-string interface instead
func CovTest(argv []string) (exitCode int) {
	file := "coverage.out"
	argv = append([]string{"test"}, argv...)
	argv = append(argv, "-cover", "-coverprofile=" + file)
	exitCode = runCommand("go", argv)
	// TODO: parse the coverage
	// TODO: print the coverage
	// TODO: set success to fail when there is missing coverage
	return
}

// Find the uncovered lines given a coverage file path
func Uncovered(path string) []string {
	data, err := ioutil.ReadFile(path)
	check(err)

	lines := splitWithoutEmpty(string(data), '\n')
	if(len(lines) == 0) {
		return []string{}
	}

	// remove the initial `set: mode` line
	lines = lines[1:]

	// filter out lines that are covered (end " 0")
	lines = filter(lines, func(line string) bool { return strings.HasSuffix(line, " 0") })

	return lines
}
