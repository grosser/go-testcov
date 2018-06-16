package go_scov

import (
	"io/ioutil"
	"strings"
	"os/exec"
	"fmt"
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

func CovTest(argv []string) bool {
	file := "coverage.out"
	argv = append([]string{"go", "test"}, argv...)
	argv = append(argv, "-cover", "-coverprofile=" + file)
	success := RunCommand(argv)
	// TODO: parse the coverage
	// TODO: print the coverage
	// TODO: set success to fail when there is missing coverage
	return success
}

// run argv and return if it succeeded
// TODO: show out and err to the user as they appear
func RunCommand(argv []string) bool {
	command := argv[0]
	out, err := exec.Command(command, argv[1:]...).Output()
	fmt.Print(string(out))
	return err == nil
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
