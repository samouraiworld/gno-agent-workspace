package std

// Both declare a concrete gno type. The Go side takes the empty interface, so
// the parameter reaches the implementation as the raw gno.TypedValue.
func AnyParam(n int) string

func AnyParamLong(n int) string
