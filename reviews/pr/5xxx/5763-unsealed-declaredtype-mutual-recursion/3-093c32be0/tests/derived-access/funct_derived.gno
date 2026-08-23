package main

type T1 func(int) T2

type T2 T1


func main() {
	var b T2 = func(i int) T2 { return nil }
	println(b(1) == nil)
	println("ok")
}
