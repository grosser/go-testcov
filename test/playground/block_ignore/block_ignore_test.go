package block_ignore

import "testing"

// Only covers part of the block: the negative and zero branches.
// The "positive" and "huge" branches stay untested on purpose, but since the
// function is inside an "// untested block", go-testcov does not complain.
func TestClassify(t *testing.T) {
	cases := map[int]string{
		-5: "negative",
		0:  "zero",
	}
	for input, want := range cases {
		if got := Classify(input); got != want {
			t.Errorf("Classify(%d) = %q, want %q", input, got, want)
		}
	}
}
