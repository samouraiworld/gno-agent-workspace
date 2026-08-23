package main

import "fmt"

type T1 func(int) T2

type T2 T1


func main() {
	var b T2 = func(i int) T2 { return nil }
	fmt.Println(b(1) == nil)
	fmt.Println("ok")
}
