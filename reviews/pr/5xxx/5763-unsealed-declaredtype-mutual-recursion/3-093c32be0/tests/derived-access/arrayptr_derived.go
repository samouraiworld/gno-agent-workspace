package main

import "fmt"

type T1 [2]*T2

type T2 T1


func main() {
	var b T2
	b[0] = &T2{}
	fmt.Println(len(b), b[1] == nil)
	fmt.Println("ok")
}
