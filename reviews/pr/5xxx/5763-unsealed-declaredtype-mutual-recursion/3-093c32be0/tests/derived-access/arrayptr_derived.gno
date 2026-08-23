package main

type T1 [2]*T2

type T2 T1


func main() {
	var b T2
	b[0] = &T2{}
	println(len(b), b[1] == nil)
	println("ok")
}
