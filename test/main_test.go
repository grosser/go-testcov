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
				exitFunction = func(got int) { exitCode = got }
				defer func() { exitFunction = os.Exit }()

				withOsArgs([]string{"executable-name", "some", "arg"}, func() {
					expectCommand(
						func() int {
							main()
							return exitCode
						},
						[]interface{}{0, "go test some arg -coverprofile coverage.out\n", ""},
					)
				})
			})
		})

		It("shows version", func() {
			exitCode := -1
			exitFunction = func(got int) { exitCode = got }
			defer func() { exitFunction = os.Exit }()

			withOsArgs([]string{"executable-name", "version"}, func() {
				expectCommand(
					func() int {
						main()
						return exitCode
					},
					[]interface{}{0, version + "\n", ""},
				)
			})
		})
	})

	// TODO: use AroundEach to run everything inside of a tempdir https://github.com/onsi/ginkgo/issues/481
	Describe("runGoTestAndCheckCoverage", func() {
		runGoTestWithCoverage := func() int { return runGoTestAndCheckCoverage([]string{"hello", "world"}) }
		withFailingTestInGoPath := func(fn func()) {
			withFakeGo("echo header > coverage.out; echo foo.com/bar/baz/foo2.go:1.2,1.3 0 >> coverage.out", func() {
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
					[]interface{}{0, "go test hello world -coverprofile coverage.out\n", ""},
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
			withFakeGo("echo header > coverage.out; echo foo:1.2,1.3 1 >> coverage.out", func() {
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
			withFakeGo("echo header > coverage.out; echo foo:1.2,1.3 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{1, "", "foo new untested sections introduced (1 current vs 0 configured)\nfoo:1.2,1.3\n"},
					)
				})
			})
		})

		It("does not show generated files when failing", func() {
			withFakeGo("echo header > coverage.out; echo foo:1.2,1.3 0 >> coverage.out; echo generated.go:1.21.3 0 >> coverage.out", func() {
				writeFile("foo", "")
				writeFile("generated.go", "")
				expectCommand(
					runGoTestWithCoverage,
					[]interface{}{1, "", "foo new untested sections introduced (1 current vs 0 configured)\nfoo:1.2,1.3\n"},
				)
			})
		})

		It("ignores generated files", func() {
			withFakeGo("echo header > coverage.out; echo generated.go:1.2,1.3 0 >> coverage.out", func() {
				writeFile("generated.go", "test est")
				expectCommand(
					runGoTestWithCoverage,
					[]interface{}{0, "", ""},
				)
			})
		})

		It("fails when configured untested is below actual untested", func() {
			withFakeGo("echo header > coverage.out; echo foo:1.2,1.3 0 >> coverage.out; echo foo:2.2,2.3 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "// untested sections: 1\n")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{1, "", "foo new untested sections introduced (2 current vs 1 configured)\nfoo:1.2,1.3\nfoo:2.2,2.3\n"},
					)
				})
			})
		})

		It("fails with shortened path when in the same folder", func() {
			withFailingTestInGoPath(func() {
				expectCommand(
					runGoTestWithCoverage,
					[]interface{}{1, "", "foo2.go new untested sections introduced (1 current vs 0 configured)\nfoo2.go:1.2,1.3\n"},
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
						[]interface{}{1, "", "foo.com/bar/baz/foo2.go new untested sections introduced (1 current vs 0 configured)\nfoo.com/bar/baz/foo2.go:1.2,1.3\n"},
					)
				})
			})
		})

		It("can show untested for multiple files", func() {
			withFakeGo("echo header > coverage.out; echo foo:1.2,1.3 0 >> coverage.out; echo foo:2.2,2.3 0 >> coverage.out; echo bar:1.2,1.3 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "// untested sections: 1\n")
					writeFile(joinPath(goPath, "src", "bar"), "")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{1, "", "bar new untested sections introduced (1 current vs 0 configured)\nbar:1.2,1.3\nfoo new untested sections introduced (2 current vs 1 configured)\nfoo:1.2,1.3\nfoo:2.2,2.3\n"},
					)
				})
			})
		})

		It("keeps sections in their natural order", func() {
			withFakeGo("echo header > coverage.out; echo foo:1.2,1.3 0 >> coverage.out; echo foo:2.2,2.3 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "// untested sections: 1\n")
					writeFile(joinPath(goPath, "src", "bar"), "")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{1, "", "foo new untested sections introduced (2 current vs 1 configured)\nfoo:1.2,1.3\nfoo:2.2,2.3\n"},
					)
				})
			})
		})

		It("passes when configured untested is equal to actual untested", func() {
			withFakeGo("echo header > coverage.out; echo foo:1.2,1.3 0 >> coverage.out; echo foo:2.2,2.3 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "// untested sections: 2\n\n")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{0, "", ""},
					)
				})
			})
		})

		It("passes when configured + inline untested is equal to actual untested", func() {
			withFakeGo("echo header > coverage.out; echo foo:2.2,2.3 0 >> coverage.out; echo foo:3.2,3.3 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "// untested sections: 2\nfoo// untested section\nbar\n")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{
							0,
							"",
							"foo has less untested sections (1 current vs 2 configured), decrement configured untested?\nconfigured on: " + joinPath(goPath, "src", "foo") + ":1",
						},
					)
				})
			})
		})

		It("passes when configured via inline comments", func() {
			withFakeGo("echo header > coverage.out; echo foo:1.2,3.0 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "func main(){\n// untested section\n}")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{0, "", ""},
					)
				})
			})
		})

		It("passes when inline comment is above the section", func() {
			withFakeGo("echo header > coverage.out; echo foo:2.2,4.0 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "\t// untested section\nfunc main(){\n\n}")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{0, "", ""},
					)
				})
			})
		})

		It("passes and warns when configured untested is above actual untested", func() {
			withFakeGo("echo header > coverage.out; echo foo:1.2,1.3 0 >> coverage.out; echo foo:2.2,2.3 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "// untested sections: 3\n")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{
							0,
							"",
							"foo has less untested sections (2 current vs 3 configured), decrement configured untested?\nconfigured on: " + joinPath(goPath, "src", "foo") + ":1",
						},
					)
				})
			})
		})

		It("fails when configured untested % is below actual untested", func() {
			withFakeGo("echo header > coverage.out; echo foo:1.2,1.3 0 >> coverage.out; echo foo:2.2,2.3 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "// untested sections: 1%\n")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{
							1,
							"",
							"foo new untested sections introduced (100% current vs 1% configured)\nfoo:1.2,1.3\nfoo:2.2,2.3\n",
						},
					)
				})
			})
		})

		It("passes when configured untested % is above actual untested", func() {
			withFakeGo("echo header > coverage.out; echo foo:1.2,1.3 0 >> coverage.out; echo foo:2.2,2.3 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "// untested sections: 100%\n")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{
							0,
							"",
							"",
						},
					)
				})
			})
		})

		It("passes when configured to ignore all untested", func() {
			withFakeGo("echo header > coverage.out; echo foo:1.2,1.3 0 >> coverage.out; echo foo:2.2,2.3 0 >> coverage.out", func() {
				withFakeGoPath(func(goPath string) {
					writeFile(joinPath(goPath, "src", "foo"), "// untested sections: ignore\n")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{
							0,
							"",
							"",
						},
					)
				})
			})
		})

		It("can warn when using unmodularized path", func() {
			withFakeGo("echo header > coverage.out; echo baz.go:1.2,1.3 0 >> coverage.out; echo baz.go:2.2,2.3 0 >> coverage.out", func() {
				withoutEnv("GOPATH", func() {
					writeFile("baz.go", "// untested sections: 3\n")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{0, "", "baz.go has less untested sections (2 current vs 3 configured), decrement configured untested?\nconfigured on: baz.go:1"},
					)
				})
			})
		})

		It("can warn when not using GOPATH", func() {
			withFakeGo("echo header > coverage.out; echo github.com/foo/bar/baz.go:1.2,1.3 0 >> coverage.out; echo github.com/foo/bar/baz.go:2.2,2.3 0 >> coverage.out", func() {
				withoutEnv("GOPATH", func() {
					writeFile("baz.go", "// untested sections: 3\n")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{0, "", "baz.go has less untested sections (2 current vs 3 configured), decrement configured untested?\nconfigured on: baz.go:1"},
					)
				})
			})
		})

		It("can warn when using GOPATH but not being in GOPATH", func() {
			withFakeGo("echo header > coverage.out; echo github.com/foo/bar/baz.go:1.2,1.3 0 >> coverage.out; echo github.com/foo/bar/baz.go:2.2,2.3 0 >> coverage.out", func() {
				withEnv("GOPATH", "/foo", func() {
					writeFile("baz.go", "// untested sections: 3\n")
					expectCommand(
						runGoTestWithCoverage,
						[]interface{}{0, "", "baz.go has less untested sections (2 current vs 3 configured), decrement configured untested?\nconfigured on: baz.go:1"},
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

		It("keeps coverage.out when requested", func() {
			withFakeGo("touch coverage.out\necho 1", func() {
				expectCommand(
					func() int { return runGoTestAndCheckCoverage([]string{"hello", "world", "-cover"}) },
					[]interface{}{0, "1\n", ""},
				)
				_, err := os.Stat("coverage.out")
				Expect(err).To(BeNil())
			})
		})

		It("can run ginkgo", func() {
			withFakeExecutable("ginkgo", "touch coverage.out\necho ginkgo \"$@\"", func() {
				writeFile("foo", "")
				expectCommand(
					func() int { return runGoTestAndCheckCoverage([]string{"ginkgo", "./..."}) },
					[]interface{}{0, "ginkgo -cover -coverprofile coverage.out ./...\n", ""},
				)
			})
		})

		Context("untested block", func() {
			It("passes when configured to ignore untested block", func() {
				withFakeGo(
					// example taken from Readme.md + 1 line
					"echo header > coverage.out; echo foo:2.13,3.13 0 0 >> coverage.out; echo foo:3.13,4.4 0 0 >> coverage.out; echo foo:6.3,6.18 0 0 >> coverage.out",
					func() {
						withFakeGoPath(func(goPath string) {
							// example taken from Readme.md + 1 line
							writeFile(joinPath(goPath, "src", "foo"), "// untested block\nfunc main() {\n  if foo(1) {\n      fmt.Print(\"Hi\")\n  }\n  fmt.Print(\"Ho\")\n}")
							expectCommand(
								runGoTestWithCoverage,
								[]interface{}{0, "", ""},
							)
						})
					},
				)
			})

			It("fails when missing coverage is after configured untested block ends", func() {
				withFakeGo(
					// example taken from Readme.md + 1 line
					"echo header > coverage.out; echo foo:2.13,3.13 0 0 >> coverage.out; echo foo:3.13,4.4 0 0 >> coverage.out; echo foo:8.3,8.18 0 0 >> coverage.out",
					func() {
						withFakeGoPath(func(goPath string) {
							// example taken from Readme.md + 1 line
							writeFile(joinPath(goPath, "src", "foo"), "// untested block\nfunc main() {\n  if foo(1) {\n      fmt.Print(\"Hi\")\n  }\n  fmt.Print(\"Ho\")\n}\nuntested-here")
							expectCommand(
								runGoTestWithCoverage,
								[]interface{}{
									1,
									"",
									"foo new untested sections introduced (1 current vs 0 configured)\nfoo:8.3,8.18\n",
								},
							)
						})
					},
				)
			})

			It("warns when untested block is misconfigured", func() {
				withFakeGo(
					// example taken from Readme.md + 1 line
					"echo header > coverage.out; echo foo:2.13,3.13 0 0 >> coverage.out; echo foo:3.13,4.4 0 0 >> coverage.out; echo foo:8.3,8.18 0 0 >> coverage.out",
					func() {
						withFakeGoPath(func(goPath string) {
							// example taken from Readme.md + 1 line
							writeFile(joinPath(goPath, "src", "foo"), "\t\t\t\t\t// untested block\nfunc main() {\n  if foo(1) {\n      fmt.Print(\"Hi\")\n  }\n  fmt.Print(\"Ho\")\n}\nuntested-here")
							expectCommand(
								runGoTestWithCoverage,
								[]interface{}{
									1,
									"",
									"go-testcov: unable to find the end of the `// untested block` started between 1 and 2, a line starting with \t\t\t\t\t}foo new untested sections introduced (3 current vs 0 configured)\nfoo:2.13,3.13\nfoo:3.13,4.4\nfoo:8.3,8.18\n",
								},
							)
						})
					},
				)
			})
		})
	})

	Describe("untestedSections", func() {
		It("shows nothing for empty", func() {
			withTempFile("", func(file *os.File) {
				Expect(untestedSections(file.Name())).To(Equal([]Section{}))
			})
		})

		It("shows untested", func() {
			withTempFile("mode: set\nfoo/pkg.go:1.2,3.4 1 0\n", func(file *os.File) {
				Expect(untestedSections(file.Name())).To(Equal([]Section{{"foo/pkg.go", 1, 2, 3, 4, 100002}}))
			})
		})

		It("does not show covered", func() {
			withTempFile("mode: set\nfoo/pkg.go:1.2,3.4 1 1\n", func(file *os.File) {
				Expect(untestedSections(file.Name())).To(Equal([]Section{}))
			})
		})

		It("does not show covered even if coverage ends in 0", func() {
			withTempFile("mode: set\nfoo/pkg.go:1.2,3.4 1 10\n", func(file *os.File) {
				Expect(untestedSections(file.Name())).To(Equal([]Section{}))
			})
		})
	})

	Describe("configuredUntestedForFile", func() {
		It("returns 0,0 when not configured", func() {
			inTempDir(func() {
				writeFile(joinPath("foo"), "")
				count, percent, line := configuredUntestedForFile("foo")
				Expect(count).To(Equal(0))
				Expect(percent).To(Equal(false))
				Expect(line).To(Equal(0))
			})
		})

		It("returns number of untested and line number of comment when configured", func() {
			inTempDir(func() {
				writeFile("foo", "// untested sections: 12")
				count, percent, line := configuredUntestedForFile("foo")
				Expect(count).To(Equal(12))
				Expect(percent).To(Equal(false))
				Expect(line).To(Equal(1))
			})
		})

		It("returns number of untested and line number of comment when configured with multiple lines", func() {
			inTempDir(func() {
				writeFile("foo", "... bork ... \n // untested sections: 12 \n ... bork ...")
				count, percent, line := configuredUntestedForFile("foo")
				Expect(count).To(Equal(12))
				Expect(percent).To(Equal(false))
				Expect(line).To(Equal(2))
			})
		})

		It("returns ignored when configured", func() {
			inTempDir(func() {
				writeFile("foo", "... bork ... \n // untested sections: ignore \n ... bork ...")
				count, percent, line := configuredUntestedForFile("foo")
				Expect(count).To(Equal(100))
				Expect(percent).To(Equal(true))
				Expect(line).To(Equal(2))
			})
		})

		It("returns percent when configured", func() {
			inTempDir(func() {
				writeFile("foo", "... bork ... \n // untested sections: 10% \n ... bork ...")
				count, percent, line := configuredUntestedForFile("foo")
				Expect(count).To(Equal(10))
				Expect(percent).To(Equal(true))
				Expect(line).To(Equal(2))
			})
		})
	})
})
