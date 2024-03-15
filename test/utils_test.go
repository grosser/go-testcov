package main

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("go-testcov", func() {
	Describe("check", func() {
		It("does nothing when no error occured", func() {
			check(nil)
		})

		It("panics when an error occured", func() {
			defer func() {
				recovered := recover()
				Expect(recovered).ToNot(BeNil())
			}()
			check(os.Remove("NOPE"))
		})
	})

	Describe("runCommand", func() {
		It("runs the given command", func() {
			exitCode := -1
			stdout, stderr := captureAll(func() {
				exitCode = runCommand("echo", "123")
			})
			Expect(exitCode).To(Equal(0))
			Expect(stdout).To(Equal("123\n"))
			Expect(stderr).To(Equal(""))
		})

		It("passes on exit code when command fails", func() {
			exitCode := -1
			stdout, stderr := captureAll(func() {
				exitCode = runCommand("sh", "-c", "exit 35")
			})
			Expect(exitCode).To(Equal(35))
			Expect(stdout).To(Equal(""))
			Expect(stderr).To(Equal(""))
		})

		It("returns an error when invalid executable was used", func() {
			exitCode := -1
			stdout, stderr := captureAll(func() {
				exitCode = runCommand("wut", "--nope")
			})
			Expect(exitCode).To(Equal(1))
			Expect(stdout).To(Equal(""))
			Expect(stderr).To(Equal("Could not get exit code for failed program: wut, [--nope]\n"))
		})
	})
})
