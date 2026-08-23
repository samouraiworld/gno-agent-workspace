package main

type T2 T1

type T1 map[string]T2


func main() {
	a := T1{}
	a["x"] = T2{}
	println(len(a))
	println("ok")
}
