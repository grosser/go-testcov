// cd test/playground && ../../go-testcov ./...
package block_ignore

// untested block
func Classify(n int) string {
	if n < 0 {
		return "negative"
	} else if n == 0 {
		return "zero"
	} else if n > 1000 {
		// only reachable with large input, never exercised by the test
		return "huge"
	}
	return "positive"
}
