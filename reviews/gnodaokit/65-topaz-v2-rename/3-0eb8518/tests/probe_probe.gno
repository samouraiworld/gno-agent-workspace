// Package probe records who its caller is, as the chain sees it.
package probe

var last string

// Record stores the pkgpath of whoever crossed into this realm.
func Record(cur realm) {
	p := cur.Previous()
	if p.IsUser() {
		last = "user:" + p.Address().String()
	} else {
		last = p.PkgPath()
	}
}

func Last() string { return last }
func Reset()       { last = "" }
