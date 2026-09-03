// MAXALLOC: 500000000
package main

import "strings"

func main() {
	s := strings.Repeat("a", 1000000)
	n := 0
	for i := 0; i < 100; i++ {
		t := s[i : i+900000]
		n += len(t)
	}
	_ = n
}

// Gas:
// 0
