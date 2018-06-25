package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"testing"
	"bytes"
	"io"
)

func TestAwesome(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Example")
}

func noError(err error) {
	Expect(err).To(BeNil())
}

func writeFile(path string, content string) {
	err := ioutil.WriteFile(path, []byte(content), 0755)
	noError(err)
}

func withTempFile(content string, fn func(*os.File)) {
	file, err := ioutil.TempFile("", "go-scov")
	noError(err)
	defer os.Remove(file.Name())
	writeFile(file.Name(), content)
	fn(file)
}

func withTempDir(fn func(string)) {
	dir, err := ioutil.TempDir("", "go-scov")
	noError(err)
	defer os.RemoveAll(dir)
	fn(dir)
}

func withEnv(key string, value string, fn func()) {
	old := os.Getenv(key)
	os.Setenv(key, value)
	defer os.Setenv(key, old)
	fn()
}

func chDir(dir string, fn func()){
	old, err := os.Getwd()
	noError(err)

	err = os.Chdir(dir)
	noError(err)

	defer os.Chdir(old)

	fn()
}

func withFakeGo(content string, fn func()) {
	withTempDir(func(dir string){
		chDir(dir, func(){ // need to run somewhere else so we can run scov on itself
			withEnv("PATH", dir + ":" + os.Getenv("PATH"), func(){
				writeFile(dir + "/go", "#!/bin/sh\n" + content)
				fn()
			})
		})
	})
}

// https://stackoverflow.com/questions/10473800/in-go-how-do-i-capture-stdout-of-a-function-into-a-string
func captureStdout(fn func()) (captured string) {
	old := os.Stdout // keep backup of the real
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	// back to normal state
	w.Close()
	os.Stdout = old // restoring the real
	captured = <-outC
	return
}

func captureStderr(fn func()) (captured string) {
	old := os.Stderr // keep backup of the real
	r, w, _ := os.Pipe()
	os.Stderr = w

	fn()

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	// back to normal state
	w.Close()
	os.Stderr = old // restoring the real
	captured = <-outC
	return
}

func captureAll(fn func()) (stdout string, stderr string) {
	stdout = captureStdout(func(){
		stderr = captureStderr(fn)
	})
	return
}

var _ = Describe("go-testcov", func() {
	Describe("main", func(){
		It("exits", func(){
			withFakeGo("touch coverage.out\necho go \"$@\"", func(){
				exitCode := -1

				// fake the exit function so we can test it
				exitFunction = func(got int) {
					exitCode = got
				}
				defer func(){
					exitFunction = os.Exit
				}()

				// fake Args so we get a useful output
				old := os.Args
				os.Args = []string{"executable-name", "some", "arg"}
				defer func(){
					os.Args = old
				}()

				stdout, stderr := captureAll(func() {
					main()
				})

				Expect(exitCode).To(Equal(0))
				Expect(stdout).To(Equal("go test some arg -cover -coverprofile=coverage.out\n"))
				Expect(stderr).To(Equal(""))
			})
		})
	})

	// TODO: use AroundEach to run everything inside of a tempdir https://github.com/onsi/ginkgo/issues/481
	Describe("covTest", func(){
		// TODO: ideally don't show this as the backtrace when test fails https://github.com/onsi/ginkgo/issues/491
		testCovTest := func(args []string, expected []interface{}) {
			exitCode := -1
			stdout, stderr := captureAll(func(){
				exitCode = covTest(args)
			})
			Expect([]interface{}{exitCode, stdout, stderr}).To(Equal(expected))
		}

		It("adds coverage to passed in arguments", func(){
			withFakeGo("touch coverage.out\necho go \"$@\"", func(){
				writeFile("foo", "")
				testCovTest(
					[]string{"hello", "world"},
					[]interface{}{0, "go test hello world -cover -coverprofile=coverage.out\n", ""},
				)
			})
		})

		It("fails without adding noise", func(){
			withFakeGo("touch coverage.out\nexit 15", func(){
				writeFile("foo", "")
				testCovTest(
					[]string{"hello", "world"},
					[]interface{}{15, "", ""},
				)
			})
		})

		It("does not fail when coverage is ok", func(){
			withFakeGo("echo header > coverage.out; echo foo 1 >> coverage.out", func(){
				writeFile("foo", "")
				testCovTest(
					[]string{"hello", "world"},
					[]interface{}{0, "", ""},
				)
			})
		})

		It("removes existing coverage to avoid confusion", func(){
			withFakeGo("touch coverage.out", func(){
				writeFile("foo", "")
				writeFile("coverage.out", "head\ntest 0")
				testCovTest(
					[]string{"hello", "world"},
					[]interface{}{0, "", ""},
				)
			})
		})

		It("fail when coverage is not ok", func(){
			withFakeGo("echo header > coverage.out; echo foo 0 >> coverage.out", func(){
				writeFile("foo", "")
				testCovTest(
					[]string{"hello", "world"},
					[]interface{}{1, "", "foo new uncovered sections introduced (1 current vs 0 configured)\nfoo\n"},
				)
			})
		})

		It("fails when configured uncovered is below actual uncovered", func(){
			withFakeGo("echo header > coverage.out; echo foo 0 >> coverage.out; echo foo 0 >> coverage.out", func(){
				writeFile("foo", "// untested sections: 1")
				testCovTest(
					[]string{"hello", "world"},
					[]interface{}{1, "", "foo new uncovered sections introduced (2 current vs 1 configured)\nfoo\nfoo\n"},
				)
			})
		})

		It("can show uncovered for multiple files", func(){
			withFakeGo("echo header > coverage.out; echo foo:1 0 >> coverage.out; echo foo:2 0 >> coverage.out; echo bar:1 0 >> coverage.out", func(){
				writeFile("foo", "// untested sections: 1")
				writeFile("bar", "")
				testCovTest(
					[]string{"hello", "world"},
					[]interface{}{1, "", "foo new uncovered sections introduced (2 current vs 1 configured)\nfoo:1\nfoo:2\nbar new uncovered sections introduced (1 current vs 0 configured)\nbar:1\n"},
				)
			})
		})

		It("passes when configured uncovered is equal to actual uncovered", func(){
			withFakeGo("echo header > coverage.out; echo foo 0 >> coverage.out; echo foo 0 >> coverage.out", func(){
				writeFile("foo", "// untested sections: 2")
				testCovTest(
					[]string{"hello", "world"},
					[]interface{}{0, "", ""},
				)
			})
		})

		It("passes when configured uncovered is above actual uncovered", func(){
			withFakeGo("echo header > coverage.out; echo foo 0 >> coverage.out; echo foo 0 >> coverage.out", func(){
				writeFile("foo", "// untested sections: 3")
				testCovTest(
					[]string{"hello", "world"},
					[]interface{}{0, "", ""},
				)
			})
		})
	})

	Describe("runCommand", func(){
		It("runs the given command", func() {
			exitCode := -1
			stdout, stderr := captureAll(func(){
				exitCode = runCommand("echo", "123")
			})
			Expect(exitCode).To(Equal(0))
			Expect(stdout).To(Equal("123\n"))
			Expect(stderr).To(Equal(""))
		})

		It("passes on exit code when command fails", func() {
			exitCode := -1
			stdout, stderr := captureAll(func(){
				exitCode = runCommand("sh", "-c", "exit 35")
			})
			Expect(exitCode).To(Equal(35))
			Expect(stdout).To(Equal(""))
			Expect(stderr).To(Equal(""))
		})

		It("returns an error when invalid executable was used", func(){
			exitCode := -1
			stdout, stderr := captureAll(func(){
				exitCode = runCommand("wuuuut", "--nope")
			})
			Expect(exitCode).To(Equal(1))
			Expect(stdout).To(Equal(""))
			Expect(stderr).To(Equal("Could not get exit code for failed program: wuuuut, [--nope]\n"))
		})
	})

	Describe("uncoveredSections", func() {
		It("shows nothing for empty", func() {
			withTempFile("", func(file *os.File){
				Expect(uncoveredSections(file.Name())).To(Equal([]string{}))
			})
		})

		It("shows uncovered", func(){
			withTempFile("mode: set\nfoo/pkg.go:1.2,3.4 1 0\n", func(file *os.File){
				Expect(uncoveredSections(file.Name())).To(Equal([]string{"foo/pkg.go:1.2,3.4"}))
			})
		})

		It("does not show covered", func(){
			withTempFile("mode: set\nfoo/pkg.go:1.2,3.4 1 1\n", func(file *os.File){
				Expect(uncoveredSections(file.Name())).To(Equal([]string{}))
			})
		})

		It("does not show covered even if coverage ends in 0", func(){
			withTempFile("mode: set\nfoo/pkg.go:1.2,3.4 1 10\n", func(file *os.File){
				Expect(uncoveredSections(file.Name())).To(Equal([]string{}))
			})
		})
	})

	Describe("configuredUncovered", func(){
		It("returns 0 when not configured", func(){
			withTempFile("", func(file *os.File){
				Expect(configuredUncovered(file.Name())).To(Equal(0))
			})
		})

		It("returns number when configured", func(){
			withTempFile("// untested sections: 12", func(file *os.File){
				Expect(configuredUncovered(file.Name())).To(Equal(12))
			})
		})
	})

	Describe("check", func(){
		It("does nothing when no error occured", func(){
			check(nil)
		})

		It("panics when an error occured", func(){
			defer func() {
				recovered := recover()
				Expect(recovered).ToNot(BeNil())
			}()
			check(os.Remove("NOPE"))
		})
	})
})
