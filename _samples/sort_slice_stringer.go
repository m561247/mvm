package main

import (
	"fmt"
	"sort"
)

// sort.Slice takes the slice as `any` and reorders it via reflect.Swapper, so
// the raw slice -- not a display wrapper -- must reach it even when the element
// type defines String() (otherwise: "reflect: call of Swapper on ptr Value").
// Printing the sorted slice then renders each element through String(), reading
// its unexported fields without tripping reflect's read-only check.

type person struct {
	name string
	age  int
}

func (p person) String() string { return fmt.Sprintf("%s:%d", p.name, p.age) }

func main() {
	people := []person{{"Bob", 31}, {"Al", 17}, {"Cy", 24}}
	sort.Slice(people, func(i, j int) bool { return people[i].age < people[j].age })
	fmt.Println(people)
}

// Output:
// [Al:17 Cy:24 Bob:31]
