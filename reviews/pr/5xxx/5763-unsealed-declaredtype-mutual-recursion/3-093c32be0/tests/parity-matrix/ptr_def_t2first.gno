package main

type T2 T1

type T1 *T2


func main() {
	var a T1
	println(a == nil)
	println("ok")
}
