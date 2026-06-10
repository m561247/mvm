package main

import (
	"fmt"
	"reflect"
)

type renamedUint8 uint8
type myString string
type cfunc func(int) int

func gte(x int) int { return x + 1 }

var ops = map[string]cfunc{">=": gte}

var g myString = "global"

func main() {
	barray := [3]renamedUint8{1, 2, 3}
	fmt.Printf("%#v %#v\n", barray, barray[:])
	fmt.Printf("%T %T\n", renamedUint8(3), myString("x"))
	m := map[myString]int{"a": 1}
	var v myString = "lit"
	v += "+more"
	fmt.Println(g, m, v, v == "lit+more")
	var f cfunc = gte
	fmt.Println(reflect.ValueOf(f).Type(), reflect.ValueOf(f) == reflect.ValueOf(ops[">="]))
}

// Output:
// [3]main.renamedUint8{0x1, 0x2, 0x3} []main.renamedUint8{0x1, 0x2, 0x3}
// main.renamedUint8 main.myString
// global map[a:1] lit+more true
// main.cfunc true
