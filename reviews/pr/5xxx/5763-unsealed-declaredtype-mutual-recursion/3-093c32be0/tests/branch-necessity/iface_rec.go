package main

import "fmt"

type T1 interface {
	M() *T2
}

type T2 T1

type impl struct{ v int }

func (i impl) M() *T2 { return nil }

func main() {
	var b T2 = impl{5}
	fmt.Println(b.M() == nil)
	fmt.Println("ok")
}
