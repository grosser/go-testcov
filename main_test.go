package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
)

var _ = Describe("go-testcov", func() {
	Describe("main", func() {
		It("exits", func() {
			withFakeGo("touch coverage.out\necho go \"$@\"", func() {
				exitCode := -1

				// fake the exit function so we can test it
				exitFunction = func(got int) {
					exitCode = got
				}
				defer func() {
					exitFunction = os.Exit
				}()

				withOsArgs([]string{"executable-name", "some", "arg"}, func() {
					expectCommand(
						func() int {
							main()
							return exitCode
						},
						[]interface{}{0, "go test some arg -cover -coverprofile=coverage.out\n", ""},
					)
				})
			})
		})
	})

	// TODO: use AroundEach to run everything inside of a tempdir https://github.com/onsi/ginkgo/issues/481
	Describe("goTestCheckCoverage", func() {
		runGoTestWithCoverage := func() int { return goTestCheckCoverage([]string{"hello", "world"}) }
		withFailingTestInGoPath := func(fn func()) {
			withFakeGo("echo header > coverage.out; echo foo.com/bar/baz/foo2.go:123 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					dir := joinPath(goPath, "src", "foo.com", "bar", "baz")
					os.MkdirAll(dir, 0700)
					writeFile(joinPath(dir, "foo2.go"), "")
					chDir(dir, fn)
				})
			})
		}

		It("adds coverage to passed in arguments", func() {
			withFakeGo("touch coverage.out\necho go \"$@\"", func() {
				writeFile("foo", "")
				expectCommand(
					runGoTestWithCoverage,
					[]interface{}{0, "go test hello world -cover -coverprofile=coverage.out\n", ""},
				)
			})
		})

		It("fails without adding noise", func() {
			withFakeGo("touch coverage.out\nexit 15", func() {
				writeFile("foo", "")
				expectCommand(
					runGoTestWithCoverage,
					[]interface{}{15, "", ""},
				)
			})
		})

		It("does not fail when coverage is ok", func() {
			withFakeGo("echo header > coverage.out; echo foo 1 >> coverage.out", func() {
				writeFile("foo", "")
				expectCommand(
					runGoTestWithCoverage,
					[]interface{}{0, "", ""},
				)
			})
		})

		It("removes existing coverage to avoid confusion", func() {
			withFakeGo("touch coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "")
					writeFile("coverage.out", "head\ntest 0")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{0, "", ""},
					)
				})
			})
		})

		It("fail when coverage is not ok", func() {
			withFakeGo("echo header > coverage.out; echo foo 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{1, "", "foo new uncovered sections introduced (1 current vs 0 configured)\nfoo\n"},
					)
				})
			})
		})

		It("fails when configured uncovered is below actual uncovered", func() {
			withFakeGo("echo header > coverage.out; echo foo 0 >> coverage.out; echo foo 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "// untested sections: 1")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{1, "", "foo new uncovered sections introduced (2 current vs 1 configured)\nfoo\nfoo\n"},
					)
				})
			})
		})

		It("fails with shortened path when in the same folder", func() {
			withFailingTestInGoPath(func() {
				expectCommand(
					runGoTestWithCoverage,
					[]interface{}{1, "", "foo2.go new uncovered sections introduced (1 current vs 0 configured)\nfoo2.go:123\n"},
				)
			})
		})

		It("fails with long path when in a different, but nested folder", func() {
			withFailingTestInGoPath(func() {
				other := joinPath(os.Getenv("GOPATH"), "src", "foo.com", "nope", "baz")
				os.MkdirAll(other, 0700)
				chDir(other, func() {
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{1, "", "foo.com/bar/baz/foo2.go new uncovered sections introduced (1 current vs 0 configured)\nfoo.com/bar/baz/foo2.go:123\n"},
					)
				})
			})
		})

		It("can show uncovered for multiple files", func() {
			withFakeGo("echo header > coverage.out; echo foo:1 0 >> coverage.out; echo foo:2 0 >> coverage.out; echo bar:1 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "// untested sections: 1")
					writeFile(joinPath(goPath, "src", "bar"), "")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{1, "", "bar new uncovered sections introduced (1 current vs 0 configured)\nbar:1\nfoo new uncovered sections introduced (2 current vs 1 configured)\nfoo:1\nfoo:2\n"},
					)
				})
			})
		})

		It("keeps sections in their natural order", func() {
			withFakeGo("echo header > coverage.out; echo foo:2 0 >> coverage.out; echo foo:10 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "// untested sections: 1")
					writeFile(joinPath(goPath, "src", "bar"), "")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{1, "", "foo new uncovered sections introduced (2 current vs 1 configured)\nfoo:2\nfoo:10\n"},
					)
				})
			})
		})

		It("passes when configured uncovered is equal to actual uncovered", func() {
			withFakeGo("echo header > coverage.out; echo foo 0 >> coverage.out; echo foo 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "// untested sections: 2")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{0, "", ""},
					)
				})
			})
		})

		It("passes when configured uncovered is above actual uncovered", func() {
			withFakeGo("echo header > coverage.out; echo foo 0 >> coverage.out; echo foo 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "// untested sections: 3")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{0, "", ""},
					)
				})
			})
		})

		It("cleans up coverage.out", func() {
			withFakeGo("touch coverage.out\necho 1", func() {
				expectCommand(
					runGoTestWithCoverage,
					[]interface{}{0, "1\n", ""},
				)
				_, err := os.Stat("coverage.out")
				Expect(err).ToNot(BeNil())
			})
		})
	})

	Describe("uncoveredSections", func() {
		It("shows nothing for empty", func() {
			withTempFile("", func(file *os.File) {
				Expect(uncoveredSections(file.Name())).To(Equal([]string{}))
			})
		})

		It("shows uncovered", func() {
			withTempFile("mode: set\nfoo/pkg.go:1.2,3.4 1 0\n", func(file *os.File) {
				Expect(uncoveredSections(file.Name())).To(Equal([]string{"foo/pkg.go:1.2,3.4"}))
			})
		})

		It("does not show covered", func() {
			withTempFile("mode: set\nfoo/pkg.go:1.2,3.4 1 1\n", func(file *os.File) {
				Expect(uncoveredSections(file.Name())).To(Equal([]string{}))
			})
		})

		It("does not show covered even if coverage ends in 0", func() {
			withTempFile("mode: set\nfoo/pkg.go:1.2,3.4 1 10\n", func(file *os.File) {
				Expect(uncoveredSections(file.Name())).To(Equal([]string{}))
			})
		})
	})

	Describe("configuredUncovered", func() {
		It("returns 0 when not configured", func() {
			withFakeGoPath(func(goPath string) {
				writeFile(joinPath(goPath, "src", "foo"), "")
				Expect(configuredUncovered("foo")).To(Equal(0))
			})
		})

		It("returns number when configured", func() {
			withFakeGoPath(func(goPath string) {
				writeFile(joinPath(goPath, "src", "foo"), "// untested sections: 12")
				Expect(configuredUncovered("foo")).To(Equal(12))
			})
		})
	})
})
