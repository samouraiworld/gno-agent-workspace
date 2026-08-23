package main

type T1 *T2

type T2 T1

func main() {
	var v T2
	p := T2(&v)
	println(*p == nil)
	println("ok")
}
