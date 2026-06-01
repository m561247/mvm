package main

// A labeled break/continue that jumps out of an enclosing range loop must run
// that range's Stop, freeing the Pull/Pull2 iterator's operand-stack slots (and
// stopping its coroutine). Otherwise each crossed iteration leaks ~5 slots and
// the stack pointer climbs until it overflows. The outer-body `:=` is what made
// the leak surface as an index-out-of-range panic (it pushes at the climbed sp);
// inside a goroutine the swallowed panic instead showed up as a deadlock hang
// (this is the root cause of the `mvm test sync/atomic` hang in TestValueConcurrent).

func main() {
	xs := []any{1, 2, 3, 4}

	// continue out of a range, many iterations: previously panicked ~iter 50.
	contHits := 0
loop:
	for j := 0; j < 1000; j++ {
		y := j
		_ = y
		for _, x := range xs {
			if x == 1 {
				contHits++
				continue loop
			}
		}
	}
	println("continue", contHits)

	// break out of a range across an intervening switch.
	breakHits := 0
outer:
	for j := 0; j < 1000; j++ {
		z := j
		_ = z
		switch j % 2 {
		case 0:
			for _, x := range xs {
				if x == 3 {
					breakHits++
					break outer
				}
			}
		default:
			breakHits++
		}
	}
	println("break", breakHits)

	// continue across two nested ranges: only the crossed inner ranges stop.
	deepHits := 0
top:
	for j := 0; j < 1000; j++ {
		w := j
		_ = w
		for _, a := range xs {
			for _, b := range xs {
				if a == 1 && b == 2 {
					deepHits++
					continue top
				}
			}
		}
	}
	println("deep", deepHits)
}

// Output:
// continue 1000
// break 1
// deep 1000
