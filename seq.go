package gojq

import "iter"

// ToSeq returns the [iter.Seq] of the given [Iter].
//
// Errors yielded by the iterator, including [*HaltError], are yielded as
// values. Use [ToSeq2] to yield them as the second return value.
func ToSeq(iter Iter) iter.Seq[any] {
	return func(yield func(any) bool) {
		for {
			v, ok := iter.Next()
			if !ok {
				return
			}
			if !yield(v) {
				return
			}
		}
	}
}

// ToSeq2 returns the [iter.Seq2] of the given [Iter]. Each value is yielded
// with a nil error except for errors yielded by the iterator, which are
// yielded with a nil value. Note that iteration is stopped by a [*HaltError]
// with a nil value, which is emitted by the halt function.
func ToSeq2(iter Iter) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		for {
			v, ok := iter.Next()
			if !ok {
				return
			}
			if err, ok := v.(error); ok {
				if e, ok := err.(*HaltError); ok && e.Value() == nil {
					return
				}
				if !yield(nil, err) {
					return
				}
				continue
			}
			if !yield(v, nil) {
				return
			}
		}
	}
}
