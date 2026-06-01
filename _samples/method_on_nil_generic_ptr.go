package main

// A method called on a nil-converted generic instantiation -- (*Box[int])(nil).M()
// in one expression -- must resolve the method's global slot even though the
// instantiated body is compiled after the call site.

type Box[T any] struct{ v T }

func (b *Box[T]) Tag() int { return 7 } // body doesn't deref b, so nil is fine

func main() {
	println((*Box[int])(nil).Tag()) // 7
}

// Output:
// 7
