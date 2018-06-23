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

func withTempfile(content string, fn func(*os.File)) {
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

var _ = Describe("go_scov", func() {
	Describe("covTest", func(){
		It("adds coverage to passed in arguments", func(){
			withFakeGo("touch coverage.out\necho go \"$@\"", func(){
				exitCode := -1
				stdout, stderr := captureAll(func(){
					exitCode = covTest([]string{"hello", "world"})
				})
				Expect(exitCode).To(Equal(0))
				Expect(stdout).To(Equal("go test hello world -cover -coverprofile=coverage.out\n"))
				Expect(stderr).To(Equal(""))
			})
		})

		It("fails without adding noise", func(){
			withFakeGo("touch coverage.out\nexit 15", func(){
				exitCode := -1
				stdout, stderr := captureAll(func(){
					exitCode = covTest([]string{"hello", "world"})
				})
				Expect(exitCode).To(Equal(15))
				Expect(stdout).To(Equal(""))
				Expect(stderr).To(Equal(""))
			})
		})

		It("does not fail when coverage is ok", func(){
			withFakeGo("echo header > coverage.out; echo foo 1 >> coverage.out", func(){
				exitCode := -1
				stdout, stderr := captureAll(func(){
					exitCode = covTest([]string{"hello", "world"})
				})
				Expect(exitCode).To(Equal(0))
				Expect(stdout).To(Equal(""))
				Expect(stderr).To(Equal(""))
			})
		})

		It("removes existing coverage to avoid confusion", func(){
			withFakeGo("touch coverage.out", func(){
				writeFile("coverage.out", "head\ntest 0")
				exitCode := -1
				stdout, stderr := captureAll(func(){
					exitCode = covTest([]string{"hello", "world"})
				})
				Expect(exitCode).To(Equal(0))
				Expect(stdout).To(Equal(""))
				Expect(stderr).To(Equal(""))
			})
		})

		It("fail when coverage is not ok", func(){
			withFakeGo("echo header > coverage.out; echo foo 0 >> coverage.out", func(){
				exitCode := -1
				stdout, stderr := captureAll(func(){
					exitCode = covTest([]string{"hello", "world"})
				})
				Expect(exitCode).To(Equal(1))
				Expect(stdout).To(Equal(""))
				Expect(stderr).To(Equal("Uncovered sections found:\nfoo\n"))
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

		It("fails when command fails", func() {
			exitCode := -1
			stdout, stderr := captureAll(func(){
				exitCode = runCommand("ls", "--nope")
			})
			Expect(exitCode).To(Equal(1))
			Expect(stdout).To(Equal(""))
			Expect(stderr).To(ContainSubstring("illegal option"))
		})
	})

	Describe("uncovered", func() {
		It("shows nothing for empty", func() {
			withTempfile("", func(file *os.File){
				Expect(uncovered(file.Name())).To(Equal([]string{}))
			})
		})

		It("shows uncovered", func(){
			withTempfile("mode: set\nfoo/pkg.go:1.2,3.4 1 0\n", func(file *os.File){
				Expect(uncovered(file.Name())).To(Equal([]string{"foo/pkg.go:1.2,3.4"}))
			})
		})

		It("does not show covered", func(){
			withTempfile("mode: set\nfoo/pkg.go:1.2,3.4 1 1\n", func(file *os.File){
				Expect(uncovered(file.Name())).To(Equal([]string{}))
			})
		})

		It("does not show covered even if coverage ends in 0", func(){
			withTempfile("mode: set\nfoo/pkg.go:1.2,3.4 1 10\n", func(file *os.File){
				Expect(uncovered(file.Name())).To(Equal([]string{}))
			})
		})
	})
})
