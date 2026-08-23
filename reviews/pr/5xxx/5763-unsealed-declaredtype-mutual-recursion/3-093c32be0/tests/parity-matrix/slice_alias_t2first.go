package main

import "fmt"

type T2 = T1

type T1 []T2


func main() {
	var a T1
	a = append(a, T2{})
	fmt.Println(len(a))
	fmt.Println("ok")
}
