package main

import "fmt"

type T1 map[string]T2

type T2 T1


func main() {
	b := T2{}
	b["k"] = T2{"j": nil}
	fmt.Println(len(b), len(b["k"]))
	fmt.Println("ok")
}
