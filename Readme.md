# go-testcov [![Build Status](https://travis-ci.com/grosser/go-testcov.png)](https://travis-ci.com/grosser/go-testcov) ![Coverage](https://img.shields.io/badge/Coverage-100%25-green.svg)

Run `go test` but fails when there are any uncovered lines.

 - Get actionable feedback on every successful test run
 - No more PRs with bad test coverage

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
