package util

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveIntermediatePrefixes(t *testing.T) {
	orig := []string{"fred", "fred/bear", "readings", "work", "work/coffee", "work/coffee/africa"}
	unique := RemoveIntermediatePrefixes(orig, "/")

	assert.Equal(t, 3, len(unique), unique)

	assert.Contains(t, unique, "readings")
	assert.Contains(t, unique, "fred/bear")
	assert.Contains(t, unique, "work/coffee/africa")

	assert.Equal(t, 6, len(orig))
}

func BenchmarkRemoveIntermediatePrefixes(t *testing.B) {
	tests := [][]string{
		{"fred", "fred/bear", "readings", "work", "work/coffee", "work/coffee/africa"},
		{"a", "b", "c", "d"},
		{"a", "a/b", "a/b/c", "a/b/c/d"},
		{"a/b/c/d", "a/b/c", "a/b", "a"},
	}

	for i := 0; i < t.N; i++ {
		t := rand.Intn(len(tests))
		RemoveIntermediatePrefixes(tests[t], "/")
	}
}