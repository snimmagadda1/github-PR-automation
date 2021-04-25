package main

import (
	"testing"
)

func TestContains(t *testing.T) {
	repos := []string{"repo1", "repo2", "repo3", "xyz", "111"}

	var isTrue, isFalse bool
	in1 := "repo1"
	in2 := "nothere"
	isTrue = contains(repos, in1)
	isFalse = contains(repos, in2)

	if !isTrue {
		t.Errorf("Did not find %s in %v", in1, repos)
	}

	if isFalse {
		t.Errorf("Did not find %s in %v", in2, repos)
	}
}
