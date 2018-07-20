# go-testcov [![Build Status](https://travis-ci.com/grosser/go-testcov.png)](https://travis-ci.com/grosser/go-testcov) ![Coverage](https://img.shields.io/badge/Coverage-100%25-green.svg)

`go test` that fails on uncovered lines and shows them

 - 🎉 Instant, actionable feedback on 💚 test run
 - 🎉 Onboard legacy code with `// untested sections: 5` comment
 - 🚫 PRs with bad test coverage
 - 🚫 External/paid coverage tools

```
go get github.com/grosser/go-testcov
go-testcov
...
test output
...
pkg.go new uncovered sections introduced (1 current vs 0 configured)
pkg.go:20.14,21.11
pkg.go:54.5,56.5
```


## Notes

 - Details on how coverage in go works https://blog.golang.org/cover and it's limitations
 - Runtime overhead is about 3%
 - Use `-covermode atomic` when testing parallel algorithms


Author
======
[Michael Grosser](http://grosser.it)<br/>
michael@grosser.it<br/>
License: MIT<br/>
