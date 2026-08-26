// PKGPATH: gno.land/r/test/mdalias
package mdalias

import (
	"gno.land/r/demo/zprobecol"
)

func main(cur realm) {
	zprobecol.Setup(cross(cur), address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5"))

	m := zprobecol.View()
	d, err := m.TokenMetadata("1")
	if err != nil {
		panic(err)
	}
	println("read from another realm:", d.Attributes[0].Value)

	d.Attributes[0].Value = "legendary" // write into the slice the getter returned

	again, _ := m.TokenMetadata("1")
	println("stored value now:", again.Attributes[0].Value)
}
