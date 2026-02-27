package levenshtein

// Distance computes the Levenshtein edit distance between two strings.
// The implementation uses O(min(m,n)) memory.
func Distance(s, t string) int {
	sr := []rune(s)
	tr := []rune(t)

	// If either is empty, the distance is the length of the other
	if len(sr) == 0 {
		return len(tr)
	}
	if len(tr) == 0 {
		return len(sr)
	}

	// Shorter string should be the "column"
	if len(sr) > len(tr) {
		sr, tr = tr, sr
	}

	// Two rows suffice
	prev := make([]int, len(sr)+1)
	curr := make([]int, len(sr)+1)

	for i := range prev {
		prev[i] = i
	}

	for j, tc := range tr {
		curr[0] = j + 1
		for i, sc := range sr {
			cost := 1
			if sc == tc {
				cost = 0
			}
			curr[i+1] = min3(
				curr[i]+1,    // deletion
				prev[i+1]+1,  // insertion
				prev[i]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}

	return prev[len(sr)]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
