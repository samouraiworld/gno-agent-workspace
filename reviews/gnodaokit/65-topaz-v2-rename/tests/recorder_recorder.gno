package recorder

import "chain/runtime/unsafe"

var Seen []string

// SetString mimics r/demo/profile.SetStringField: a crossing setter that
// records which realm crossed into it.
func SetString(cur realm, field string, value string) bool {
	Seen = append(Seen, unsafe.PreviousRealm().PkgPath()+"|"+field+"="+value)
	return true
}

func GetString(addr address, field string, def string) string { return def }

func Get() []string { return Seen }
