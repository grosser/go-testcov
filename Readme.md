# go-testcov [![Test](https://github.com/grosser/go-testcov/actions/workflows/test.yml/badge.svg)](https://github.com/grosser/go-testcov/actions?query=branch%3Amaster) [![coverage](https://img.shields.io/badge/coverage-100%25-success.svg)](https://github.com/grosser/go-testcov)

`go test` that fails on untested lines and shows them

 - 🎉 Instant, actionable feedback on 💚 test run
 - 🎉 Onboard legacy code with top of the file `// untested sections: 5` comment 
 - 🎉 Mark untested code sections with inline `// untested section` comment
 - 🚫 PRs with bad test coverage
 - 🚫 External/paid coverage tools

```
go get github.com/grosser/go-testcov
go-testcov . # same arguments as `go test` uses, so for example `go-testcov ./...` for everything
...
test output
...
pkg.go new untested sections introduced (1 current vs 0 configured)
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

- all tests are in `test/` so the main library does not force installation of gomega + ginkgo
- the files from the root folder are symlinked there to make everything load
- easiest to work from that folder directly


Author
======
[Michael Grosser](http://grosser.it)<br/>
michael@grosser.it<br/>
License: MIT<br/>
