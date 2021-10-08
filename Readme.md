# go-testcov [![Build Status](https://travis-ci.com/grosser/go-testcov.svg)](https://travis-ci.com/grosser/go-testcov) [![coverage](https://img.shields.io/badge/coverage-100%25-success.svg)](https://github.com/grosser/go-testcov)

`go test` that fails on uncovered lines and shows them

 - 🎉 Instant, actionable feedback on 💚 test run
 - 🎉 Onboard legacy code with top of the file `// untested sections: 5` comment 
 - 🎉 Mark uncovered code sections with inline `// untested section` comment
 - 🚫 PRs with bad test coverage
 - 🚫 External/paid coverage tools

```
go get github.com/grosser/go-testcov
go-testcov . # same arguments as `go test` uses
...
test output
...
pkg.go new uncovered sections introduced (1 current vs 0 configured)
pkg.go:20.14,21.11
pkg.go:54.5,56.5
```


## Notes

 - [coverage in go](https://blog.golang.org/cover)
 - Runtime overhead for coverage is about 3%
 - Use `-covermode atomic` when testing parallel algorithms
 - To keep the `coverage.out` file run with `-cover`


## Development

Run `go-testcov` on itself:

```
go install
cd test
go-testcov
```

all tests are in `test/` so the main library does not force installation of gomega + ginkgo


Author
======
[Michael Grosser](http://grosser.it)<br/>
michael@grosser.it<br/>
License: MIT<br/>
