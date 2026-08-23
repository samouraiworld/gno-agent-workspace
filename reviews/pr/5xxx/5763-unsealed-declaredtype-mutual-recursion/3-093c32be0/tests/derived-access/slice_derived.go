package main

import "fmt"

type T1 []T2

type T2 T1


func main() {
	var b T2
	b = append(b, T2{})
	b[0] = append(b[0], T2{})
	fmt.Println(len(b), len(b[0]))
	fmt.Println("ok")
}
