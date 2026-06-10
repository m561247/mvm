package main

// A label may share its name with a variable in the same function (separate
// namespaces); the label must not hijack the variable's symbol.

func f(b bool) string {
	out := "start"
	done := false
	if b {
		goto done
	}
	out += " no-jump"
	done = true
done:
	if done {
		out += " done-true"
	}
	return out
}

func main() {
	println(f(true))
	println(f(false))
}

// Output:
// start
// start no-jump done-true
