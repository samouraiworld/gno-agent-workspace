package main

type T2 T1

type T1 interface {
	M() int
}

type impl struct{}

func (impl) M() int { return 42 }

func main() {
	var a T1 = impl{}
	println(a.M())
	println("ok")
}
