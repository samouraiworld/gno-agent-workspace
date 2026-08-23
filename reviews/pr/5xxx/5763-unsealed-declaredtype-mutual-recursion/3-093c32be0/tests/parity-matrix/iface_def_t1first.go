package main

import "fmt"

type T1 interface {
	M() int
}

type T2 T1

type impl struct{}

func (impl) M() int { return 42 }

func main() {
	var a T1 = impl{}
	fmt.Println(a.M())
	fmt.Println("ok")
}
