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
	path      string
	startLine int
	startChar int
	endLine   int
	endChar   int
	sortValue int
}

var inlineIgnore = regexp.MustCompile("//.*untested section(\\s|,|$)")
var startInlineIgnore = regexp.MustCompile("^\\s*//.*untested section(\\s|,|$)")
var generatedFile = regexp.MustCompile("/*generated.*\\.go$")

// covert raw coverage line into a section github.com/foo/bar/baz.go:1.2,3.5 1 0
func NewSection(raw string) Section {
	parts := strings.SplitN(raw, ":", 2)
	file := parts[0]
	parts = strings.FieldsFunc(parts[1], func(r rune) bool { return r == '.' || r == ',' || r == ' ' })
	startLine := stringToInt(parts[0])
	startChar := stringToInt(parts[1])
	endLine := stringToInt(parts[2])
	endChar := stringToInt(parts[3])
	sortValue := startLine*100000 + startChar // we group by path, so we only need to sort by line+char
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
	_ = os.Remove(coveragePath)

	// allow users to keep the coverage.out file
	// TODO: parse options to find the location the user wanted and use+keep that
	if !containsString(argv, "-cover") {
		defer os.Remove(coveragePath)
	}

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

// Tests passed, so let's check coverage for each path that has coverage
func checkCoverage(coveragePath string) (exitCode int) {
	untestedSections := untestedSections(coveragePath)
	pathSections := groupSectionsByPath(untestedSections)
	wd, err := os.Getwd()
	check(err)

	iterateSorted(pathSections, func(path string, sections []Section) {
		if generatedFile.MatchString(path) {
			return
		}
		// remove package prefix like "github.com/user/lib", but cache the call to os.Getwd
		displayPath, readPath := normalizeModulePath(path, wd)
		configured := configuredUntested(readPath)
		content := strings.Split(readFile(readPath), "\n")
		sections = filterSectionsIgnoredInline(sections, content)
		current := len(sections)

		if current == configured {
			return
		}

		details := fmt.Sprintf("(%v current vs %v configured)", current, configured)

		if current > configured {
			// TODO: color when tty
			_, _ = fmt.Fprintf(os.Stderr, "%v new untested sections introduced %v\n", displayPath, details)

			// sort sections since go does not
			sort.Slice(sections, func(i, j int) bool {
				return sections[i].sortValue < sections[j].sortValue
			})

			for _, section := range sections {
				// copy-paste friendly snippets
				_, _ = fmt.Fprintln(os.Stderr, displayPath+":"+section.Numbers())
			}

			exitCode = 1
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "%v has less untested sections %v, decrement configured untested?\n", displayPath, details)
		}
	})
	return
}

// keep sections that are marked with "untested section" comment
// need to be careful to not change the list while iterating, see https://pauladamsmith.com/blog/2016/07/go-modify-slice-iteration.html
// NOTE: this is a bit rough as it does not account for partial lines via start/end characters
func filterSectionsIgnoredInline(sections []Section, content []string) []Section {
	uncheckedSections := sections
	sections = []Section{}
	for _, section := range uncheckedSections {
		for lineNumber := section.startLine; lineNumber <= section.endLine; lineNumber++ {
			if inlineIgnore.MatchString(content[lineNumber-1]) {
				break // section is ignored
			} else if lineNumber >= 2 && startInlineIgnore.MatchString(content[lineNumber-2]) {
				break // section is ignored by inline ignore above it
			} else if lineNumber == section.endLine {
				sections = append(sections, section) // keep the section
			}
		}
	}
	return sections
}

func groupSectionsByPath(sections []Section) (grouped map[string][]Section) {
	grouped = map[string][]Section{}
	for _, section := range sections {
		path := section.path
		group, ok := grouped[path]
		if !ok {
			grouped[path] = []Section{}
		}
		grouped[path] = append(group, section)
	}
	return
}

// Find the untested sections given a coverage path
func untestedSections(coverageFilePath string) (sections []Section) {
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

func normalizeModulePath(path string, workingDirectory string) (displayPath string, readPath string) {
	separator := string(os.PathSeparator)
	parts := strings.Split(path, separator)
	modulePrefixSize := len(parts) - 1
	goPath, hasGoPath := os.LookupEnv("GOPATH")
	inGoPath := false
	goPrefixedPath := joinPath(goPath, "src", path)

	if hasGoPath {
		_, err := os.Stat(goPrefixedPath)
		inGoPath = !os.IsNotExist(err)
	}

	// path too short, return a good guess
	if len(parts) <= modulePrefixSize {
		if inGoPath {
			return path, goPrefixedPath
		} else {
			return path, path
		}

	}

	prefix := strings.Join(parts[:modulePrefixSize], separator)
	demodularized := strings.SplitN(path, prefix+separator, 2)[1]

	// folder is not in go path ... remove module nesting
	if !inGoPath {
		return demodularized, demodularized
	}

	// we are in a nested folder ... remove module nesting and expand full goPath
	if strings.HasSuffix(workingDirectory, prefix) {
		return demodularized, goPrefixedPath
	}

	// testing remote package, don't expand display but expand full goPath
	return path, goPrefixedPath
}

// How many sections are expected to be untested, 0 if not configured
// TODO: return an error when the file does not exist and handle that gracefully in the caller
func configuredUntested(path string) (count int) {
	content := readFile(path)
	regex := regexp.MustCompile("// *untested sections: *([0-9]+)")
	match := regex.FindStringSubmatch(content)
	if len(match) == 2 {
		return stringToInt(match[1])
	} else {
		return 0
	}
}
