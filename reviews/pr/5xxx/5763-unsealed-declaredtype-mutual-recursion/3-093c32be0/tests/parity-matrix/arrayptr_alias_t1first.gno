package main

type T1 [2]*T2

type T2 = T1


func main() {
	var a T1
	a[0] = &T2{}
	println(len(a))
	println("ok")
}
