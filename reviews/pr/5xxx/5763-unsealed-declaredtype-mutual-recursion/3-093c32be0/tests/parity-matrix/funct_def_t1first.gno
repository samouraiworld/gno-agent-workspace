package main

type T1 func(int) T2

type T2 T1


func main() {
	var a T1
	a = func(i int) T2 { return nil }
	println(a(1) == nil)
	println("ok")
}
