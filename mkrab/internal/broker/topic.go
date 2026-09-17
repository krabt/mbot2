package broker

import "strings"

func topicFiltersOverlap(left, right string) bool {
	a, b := strings.Split(left, "/"), strings.Split(right, "/")
	for i := 0; ; i++ {
		if i < len(a) && a[i] == "#" || i < len(b) && b[i] == "#" {
			return true
		}
		if i == len(a) || i == len(b) {
			return i == len(a) && i == len(b)
		}
		if a[i] != "+" && b[i] != "+" && a[i] != b[i] {
			return false
		}
	}
}

func topicCoveredBy(allowed, requested string) bool {
	a, r := strings.Split(allowed, "/"), strings.Split(requested, "/")
	for i, part := range a {
		if part == "#" {
			return true
		}
		if i >= len(r) {
			return false
		}
		if part == "+" {
			if r[i] == "#" {
				return false
			}
			continue
		}
		if r[i] == "+" || r[i] == "#" || part != r[i] {
			return false
		}
	}
	return len(a) == len(r)
}
