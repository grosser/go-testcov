package main

import (
	"bytes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io"
	"io/ioutil"
	"os"
	"testing"
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

func chDir(dir string, fn func()) {
	old, err := os.Getwd()
	noError(err)

	err = os.Chdir(dir)
	noError(err)

	defer os.Chdir(old)

	fn()
}

func withFakeGo(content string, fn func()) {
	withTempDir(func(dir string) {
		chDir(dir, func() { // need to run somewhere else so we can run scov on itself
			withEnv("PATH", dir+":"+os.Getenv("PATH"), func() {
				writeFile(dir+"/go", "#!/bin/sh\n"+content)
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
	stdout = captureStdout(func() {
		stderr = captureStderr(fn)
	})
	return
}

func expectCommand(fn func() int, expected []interface{}) {
	exitCode := -1
	stdout, stderr := captureAll(func() {
		exitCode = fn()
	})
	ExpectWithOffset(1, []interface{}{exitCode, stdout, stderr}).To(Equal(expected))
}

func withOsArgs(args []string, fn func()) {
	old := os.Args
	os.Args = args
	defer func() {
		os.Args = old
	}()
	fn()

}
