package main

import "fmt"

type T2 T1

type T1 *T2


func main() {
	var a T1
	fmt.Println(a == nil)
	fmt.Println("ok")
}
