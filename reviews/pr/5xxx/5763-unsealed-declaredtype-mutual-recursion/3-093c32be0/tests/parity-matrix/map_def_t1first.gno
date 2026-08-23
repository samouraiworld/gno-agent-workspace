package main

type T1 map[string]T2

type T2 T1


func main() {
	a := T1{}
	a["x"] = T2{}
	println(len(a))
	println("ok")
}
