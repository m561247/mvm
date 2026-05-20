package main

import "fmt"

// A named-basic type with a String method, accessed as a struct FIELD (or
// copied out of one), must still dispatch String() under fmt's value verbs and
// satisfy interfaces (type assertion). The field's mvm type is a shallow copy
// whose own Methods are empty; the methods live on its Base back-link, which
// both interface bridging and method/interface resolution must follow (it
// previously only checked the copy, so o.Weight printed 290 not 290g and failed
// an `interface{}.(fmt.Stringer)` assertion).

type Grams int

func (g Grams) String() string { return fmt.Sprintf("%dg", int(g)) }

type Organ struct {
	Name   string
	Weight Grams
}

func main() {
	o := Organ{"heart", 290}
	fmt.Printf("%v\n", o.Weight)
	w := o.Weight
	fmt.Println(w)
	fmt.Printf("%-8s (%v)\n", o.Name, o.Weight)
	var i any = o.Weight
	_, ok := i.(fmt.Stringer)
	fmt.Println("is stringer:", ok)
}

// Output:
// 290g
// 290g
// heart    (290g)
// is stringer: true
