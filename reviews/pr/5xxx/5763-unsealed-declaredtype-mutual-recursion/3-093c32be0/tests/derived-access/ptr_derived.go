package main

import "fmt"

type T1 *T2

type T2 T1


func main() {
	var b T2
	fmt.Println(b == nil)
	fmt.Println("ok")
}
