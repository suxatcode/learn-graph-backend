package db

// All returns true if all entries of `ar` return true from the predicate `pred`
func All[T any, A ~[]T](ar A, pred func(T) bool) bool {
	for _, a := range ar {
		if !pred(a) {
			return false
		}
	}
	return true
}

// FindFirst finds the first T that satisfies predicate `pred` in `ar`, returns
// nil if none is found
func FindFirst[T any, A ~[]T](ar A, pred func(t T) bool) *T {
	for _, t := range ar {
		if pred(t) {
			return &t
		}
	}
	return nil
}

// FindFirst finds the first T that satisfies predicate `pred` in `ar` and
// returns it's index, returns -1 if none is found
func FindFirstIndex[T any, A ~[]T](ar A, pred func(t T) bool) int {
	for i, t := range ar {
		if pred(t) {
			return i
		}
	}
	return -1
}

// FindAll finds all T's that satisfies predicate `pred` in `ar`, if none is
// found an empty slice is returned (not nil!)
func FindAll[T any, A ~[]T](ar A, pred func(t T) bool) []T {
	ts := []T{}
	for _, t := range ar {
		if pred(t) {
			ts = append(ts, t)
		}
	}
	return ts
}

// ContainsP returns true if us contains v, which is accessed for each element by calling getV.
// Otherwise false is returned.
func ContainsP[U any, V comparable, Us ~[]U](us Us, v V, getV func(U) V) bool {
	for _, u := range us {
		if getV(u) == v {
			return true
		}
	}
	return false
}

// ContainsP returns true if vs contains v.
// Otherwise false is returned.
func Contains[V comparable, Vs ~[]V](vs Vs, v V) bool {
	return ContainsP(vs, v, func(v V) V { return v })
}

func RemoveIf[T any, A ~[]T](ar A, pred func(t T) bool) []T {
	newar := []T{}
	for _, a := range ar {
		if !pred(a) {
			newar = append(newar, a)
		}
	}
	return newar
}

type AnyNumber interface {
	float64 | float32 |
		int | int16 | int32 | int64 |
		uint | uint16 | uint32 | uint64
}

func Sum[T AnyNumber, U any](us []U, getNum func(U) T) T {
	sum := T(0)
	for _, u := range us {
		sum += getNum(u)
	}
	return sum
}

func DeleteAt[T any](ar []T, index int) []T {
	if len(ar) <= index || index < 0 {
		return ar
	}
	return append(ar[:index], ar[index+1:]...)
}
