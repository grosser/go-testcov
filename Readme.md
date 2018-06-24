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
Uncovered sections found:
github.com/foo/bar/pkg.go:111.12,114.2
```

Author
======
[Michael Grosser](http://grosser.it)<br/>
michael@grosser.it<br/>
License: MIT<br/>
