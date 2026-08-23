package main

import "fmt"

type T2 = T1

type T1 [2]*T2


func main() {
	var a T1
	a[0] = &T2{}
	fmt.Println(len(a))
	fmt.Println("ok")
}
