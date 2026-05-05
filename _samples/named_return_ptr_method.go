package main

type T struct{ x int }

func (t *T) Set(v int) { t.x = v }

func makeT() (t T) {
	t.Set(42)
	return t
}

type Arr [4]byte

func (a *Arr) SetByte(i int, v byte) { a[i] = v }

func makeArr() (a Arr, err error) {
	a.SetByte(2, 7)
	return a, err
}

func main() {
	t := makeT()
	println(t.x)

	a, _ := makeArr()
	println(a[2])
}

// Output:
// 42
// 7
