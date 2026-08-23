package main

type T1 []T2

type T2 T1


func main() {
	var a T1
	a = append(a, T2{})
	println(len(a))
	println("ok")
}
