package main

type T1 interface {
	M() int
}

type T2 T1

type impl struct{}

func (impl) M() int { return 42 }

func main() {
	var b T2 = impl{}
	println(b.M())
	println("ok")
}
