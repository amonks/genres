package data_test

import (
	"math"
	"testing"

	"github.com/amonks/genres/data"
	"github.com/stretchr/testify/assert"
)

func TestDistance(t *testing.T) {
	a := data.Vector{"a": 1, "b": 1, "not in b": 1}
	b := data.Vector{"a": 2, "b": 2, "not in a": 3}
	assert.Equal(t, math.Sqrt(2), a.Distance(b))
}

func TestDelta(t *testing.T) {
	a := data.Vector{"a": 1, "b": 1, "not in b": 1}
	b := data.Vector{"a": 2, "b": 2, "not in a": 3}
	assert.Equal(t, data.Vector{"a": 1, "b": 1}, a.Delta(b))
}

func TestDivide(t *testing.T) {
	a := data.Vector{"a": 2, "b": 2}
	assert.Equal(t, data.Vector{"a": 1, "b": 1}, a.Divide(2))
}

func TestMultiply(t *testing.T) {
	a := data.Vector{"a": 1, "b": 1}
	assert.Equal(t, data.Vector{"a": 2, "b": 2}, a.Multiply(2))
}

func TestAdd(t *testing.T) {
	a := data.Vector{"a": 1, "b": 1, "not in b": 1}
	b := data.Vector{"a": 2, "b": 2, "not in a": 2}
	assert.Equal(t, data.Vector{"a": 3, "b": 3, "not in b": 1}, a.Add(b))
}

func TestPath(t *testing.T) {
	a := data.Vector{"a": 1, "b": 1}
	delta := data.Vector{"a": 3, "b": 3, "not in a": 5}
	expect := []data.Vector{
		{"a": 2, "b": 2},
		{"a": 3, "b": 3},
		{"a": 4, "b": 4},
	}
	assert.Equal(t, expect, a.Path(delta, 3))
}
