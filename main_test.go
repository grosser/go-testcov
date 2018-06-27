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
				writeFile("foo", "")
				writeFile("coverage.out", "head\ntest 0")
				expectCommand(
					runGoTestWithCoverage,
					[]interface{}{0, "", ""},
				)
			})
		})

		It("fail when coverage is not ok", func() {
			withFakeGo("echo header > coverage.out; echo foo 0 >> coverage.out", func() {
				writeFile("foo", "")
				expectCommand(
					runGoTestWithCoverage,
					[]interface{}{1, "", "foo new uncovered sections introduced (1 current vs 0 configured)\nfoo\n"},
				)
			})
		})

		It("fails when configured uncovered is below actual uncovered", func() {
			withFakeGo("echo header > coverage.out; echo foo 0 >> coverage.out; echo foo 0 >> coverage.out", func() {
				writeFile("foo", "// untested sections: 1")
				expectCommand(
					runGoTestWithCoverage,
					[]interface{}{1, "", "foo new uncovered sections introduced (2 current vs 1 configured)\nfoo\nfoo\n"},
				)
			})
		})

		It("can show uncovered for multiple files", func() {
			withFakeGo("echo header > coverage.out; echo foo:1 0 >> coverage.out; echo foo:2 0 >> coverage.out; echo bar:1 0 >> coverage.out", func() {
				writeFile("foo", "// untested sections: 1")
				writeFile("bar", "")
				expectCommand(
					runGoTestWithCoverage,
					[]interface{}{1, "", "bar new uncovered sections introduced (1 current vs 0 configured)\nbar:1\nfoo new uncovered sections introduced (2 current vs 1 configured)\nfoo:1\nfoo:2\n"},
				)
			})
		})

		It("keeps sections in their natural order", func() {
			withFakeGo("echo header > coverage.out; echo foo:2 0 >> coverage.out; echo foo:10 0 >> coverage.out", func() {
				writeFile("foo", "// untested sections: 1")
				writeFile("bar", "")
				expectCommand(
					runGoTestWithCoverage,
					[]interface{}{1, "", "foo new uncovered sections introduced (2 current vs 1 configured)\nfoo:2\nfoo:10\n"},
				)
			})
		})

		It("passes when configured uncovered is equal to actual uncovered", func() {
			withFakeGo("echo header > coverage.out; echo foo 0 >> coverage.out; echo foo 0 >> coverage.out", func() {
				writeFile("foo", "// untested sections: 2")
				expectCommand(
					runGoTestWithCoverage,
					[]interface{}{0, "", ""},
				)
			})
		})

		It("passes when configured uncovered is above actual uncovered", func() {
			withFakeGo("echo header > coverage.out; echo foo 0 >> coverage.out; echo foo 0 >> coverage.out", func() {
				writeFile("foo", "// untested sections: 3")
				expectCommand(
					runGoTestWithCoverage,
					[]interface{}{0, "", ""},
				)
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

		It("removes cwd", func() {
			withTempFile("mode: set\ngithub.com/grosser/go-testcov/pkg.go:1.2,3.4 1 0\n", func(file *os.File) {
				Expect(uncoveredSections(file.Name())).To(Equal([]string{"pkg.go:1.2,3.4"}))
			})
		})

		It("does not remove non-cwd which would break file reading", func() {
			withTempFile("mode: set\ngithub.com/nope/nope/pkg.go:1.2,3.4 1 0\n", func(file *os.File) {
				Expect(uncoveredSections(file.Name())).To(Equal([]string{"github.com/nope/nope/pkg.go:1.2,3.4"}))
			})
		})
	})

	Describe("configuredUncovered", func() {
		It("returns 0 when not configured", func() {
			withTempFile("", func(file *os.File) {
				Expect(configuredUncovered(file.Name())).To(Equal(0))
			})
		})

		It("returns number when configured", func() {
			withTempFile("// untested sections: 12", func(file *os.File) {
				Expect(configuredUncovered(file.Name())).To(Equal(12))
			})
		})
	})
})
