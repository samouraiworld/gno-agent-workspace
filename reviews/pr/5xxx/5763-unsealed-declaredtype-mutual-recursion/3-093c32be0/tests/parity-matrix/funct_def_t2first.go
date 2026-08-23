package main

import "fmt"

type T2 T1

type T1 func(int) T2


func main() {
	var a T1
	a = func(i int) T2 { return nil }
	fmt.Println(a(1) == nil)
	fmt.Println("ok")
}
