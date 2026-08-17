// Package fuzzy provides simple string-distance based ranking, used to
// power typo detection ("Did you mean...") and result ordering.
package fuzzy

import "strings"

// Distance computes the Levenshtein edit distance between a and b.
func Distance(a, b string) int {
	a, b = strings.ToLower(a), strings.ToLower(b)
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)

	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = min3(del, ins, sub)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

// Scored pairs an arbitrary item with its distance-based score against a
// query, used for ranking.
type Scored[T any] struct {
	Item  T
	Score int
}

// RankByName scores each item by edit distance between name(item) and
// query, ascending (closer matches first).
func RankByName[T any](query string, items []T, name func(T) string) []Scored[T] {
	out := make([]Scored[T], len(items))
	for i, it := range items {
		out[i] = Scored[T]{Item: it, Score: Distance(query, name(it))}
	}
	// simple insertion sort — result sets are small (search API caps results)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Score < out[j-1].Score; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
