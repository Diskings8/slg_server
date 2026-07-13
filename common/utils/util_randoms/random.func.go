package util_randoms

func BetweenInt64(minValue, maxValue int64) int64 {
	if minValue >= maxValue {
		return maxValue
	}

	r := Rand()
	v := r.Int64N(maxValue-minValue) + minValue
	Release(r)
	return v
}
