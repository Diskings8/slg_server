package s2s

func MapKey2Slice[M comparable](m map[M]struct{}) []M {
	ret := make([]M, 0, len(m))
	for k := range m {
		ret = append(ret, k)
	}
	return ret
}
