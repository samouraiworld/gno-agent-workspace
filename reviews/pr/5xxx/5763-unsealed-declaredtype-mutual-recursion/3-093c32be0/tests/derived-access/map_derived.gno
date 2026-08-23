package main

type T1 map[string]T2

type T2 T1


func main() {
	b := T2{}
	b["k"] = T2{"j": nil}
	println(len(b), len(b["k"]))
	println("ok")
}
