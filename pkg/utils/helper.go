package utils

import (
	"sort"
	"strings"
)

// Contains searches a slice for a string
func Contains(s []string, searchterm string) bool {
	i := sort.SearchStrings(s, searchterm)
	contains := i < len(s) && s[i] == searchterm
	return contains
}

// GetEnvAsSlice splits an env var by a delimter and maps to slice
func GetEnvAsSlice(in string, sep string) []string {
	val := strings.Split(in, sep)

	return val
}
