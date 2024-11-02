package client

type Weighted struct {
	Server          string
	Weight          int
	CurrentWeight   int
	EffectiveWeight int
}

//	a  b  c
//	0  0  0  (initial state)
//
//	5  1  1  (a selected)
//
// -2  1  1
//
//	3  2  2  (a selected)
//
// -4  2  2
//
//	1  3  3  (b selected)
//	1 -4  3
//
//	6 -3  4  (a selected)
//
// -1 -3  4
//
//	4 -2  5  (c selected)
//	4 -2 -2
//
//	9 -1 -1  (a selected)
//	2 -1 -1
//
//	7  0  0  (a selected)
//	0  0  0
func nextWeighted(servers []*Weighted) (best *Weighted) {
	total := 0

	for i := 0; i < len(servers); i++ {
		w := servers[i]

		if w == nil {
			continue
		}

		w.CurrentWeight += w.EffectiveWeight
		total += w.EffectiveWeight
		if w.EffectiveWeight < w.Weight {
			w.EffectiveWeight++
		}

		if best == nil || w.CurrentWeight > best.CurrentWeight {
			best = w
		}
	}

	if best == nil {
		return nil
	}

	best.CurrentWeight -= total

	return best
}
