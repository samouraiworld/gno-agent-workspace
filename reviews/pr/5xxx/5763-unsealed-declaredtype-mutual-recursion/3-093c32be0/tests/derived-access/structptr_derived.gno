package main

type T1 struct {
	Next *T2
	Val  int
}

type T2 T1


func main() {
	var b T2
	b.Val = 7
	b.Next = &T2{Val: 8}
	println(b.Val, b.Next.Val)
	println("ok")
}
