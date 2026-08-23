package main

import "fmt"

type T2 T1

type T1 struct {
	Next *T2
	Val  int
}


func main() {
	var a T1
	a.Next = &T2{Val: 9}
	fmt.Println(a.Next.Val)
	fmt.Println("ok")
}
