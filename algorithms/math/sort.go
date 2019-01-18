package math

import "sort"

// SortedKeys returns sorted keys of the given dictionary
func SortedKeys(d map[string]string) []string {
	var keysSorted []string
	for k := range d {
		keysSorted = append(keysSorted, k)
	}
	sort.Strings(keysSorted)
	return keysSorted
}
