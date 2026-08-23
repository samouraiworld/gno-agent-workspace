package main

import "fmt"

type T1 interface {
	M() int
}

type T2 T1

type impl struct{}

func (impl) M() int { return 42 }

func main() {
	var b T2 = impl{}
	fmt.Println(b.M())
	fmt.Println("ok")
}
