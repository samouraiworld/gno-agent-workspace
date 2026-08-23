package main

type T1 *T2

type T2 T1


func main() {
	var a T1
	println(a == nil)
	println("ok")
}
