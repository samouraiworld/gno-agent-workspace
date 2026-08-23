package main

type T1 *T2

type T2 T1


func main() {
	var b T2
	println(b == nil)
	println("ok")
}
