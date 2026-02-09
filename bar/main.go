package main

import "os"

// some other stuff
// even more
// untested section
func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
