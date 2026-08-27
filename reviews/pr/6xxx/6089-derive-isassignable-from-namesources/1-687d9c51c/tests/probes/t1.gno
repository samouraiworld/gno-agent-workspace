package main

func f() {}

func main() {
	f = nil
	println("reached")
}
