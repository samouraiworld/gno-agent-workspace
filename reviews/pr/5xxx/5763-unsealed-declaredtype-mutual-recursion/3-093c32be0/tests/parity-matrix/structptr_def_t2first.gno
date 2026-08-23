package main

type T2 T1

type T1 struct {
	Next *T2
	Val  int
}


func main() {
	var a T1
	a.Next = &T2{Val: 9}
	println(a.Next.Val)
	println("ok")
}
