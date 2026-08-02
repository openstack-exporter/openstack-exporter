package utils

import "slices"

// RemoveElements returns a copy of slice without any elements present in drop.
func RemoveElements[S ~[]E, E comparable](slice S, drop S) S {
	res := make(S, 0, len(slice))
	for _, s := range slice {
		if slices.Contains(drop, s) {
			continue
		}
		res = append(res, s)
	}
	return res
}

// UniqueElements returns slice items with duplicates removed, preserving order.
func UniqueElements[S ~[]E, E comparable](slice S) S {
	seen := map[E]struct{}{}
	res := make(S, 0, len(slice))
	for _, s := range slice {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		res = append(res, s)
	}
	return res
}
