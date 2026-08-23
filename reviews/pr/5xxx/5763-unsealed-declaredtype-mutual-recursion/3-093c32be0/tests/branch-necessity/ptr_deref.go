package main

import "fmt"

type T1 *T2

type T2 T1

func main() {
	var v T2
	p := T2(&v)
	fmt.Println(*p == nil)
	fmt.Println("ok")
}
