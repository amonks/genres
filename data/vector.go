package data

import "math"

type Vector map[string]float64

func (this Vector) Distance(other Vector) float64 {
	var terms float64
	for k, v := range this {
		v2, has := other[k]
		if !has {
			continue
		}
		terms += math.Pow(v-v2, 2)
	}
	return math.Sqrt(terms)
}

func (this Vector) Delta(other Vector) Vector {
	delta := Vector{}
	for k, v := range this {
		v2, has := other[k]
		if !has {
			continue
		}
		delta[k] = v2 - v
	}
	return delta
}

func (this Vector) Divide(scalar float64) Vector {
	result := make(Vector, len(this))
	for k, v := range this {
		result[k] = v / scalar
	}
	return result
}

func (this Vector) Multiply(scalar float64) Vector {
	result := make(Vector, len(this))
	for k, v := range this {
		result[k] = v * scalar
	}
	return result
}

func (this Vector) Add(delta Vector) Vector {
	result := make(Vector, len(this))
	for k, v := range this {
		result[k] = v + delta[k]
	}
	return result
}

func (this Vector) Path(delta Vector, steps int) []Vector {
	increment := delta.Divide(float64(steps))
	points := make([]Vector, steps)
	last := this
	for i := 0; i < steps; i++ {
		points[i] = last.Add(increment)
		last = points[i]
	}
	return points
}
