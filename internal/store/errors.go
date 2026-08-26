package store

import "errors"

var (
	ErrNotFound = errors.New("记录不存在")
	ErrConflict = errors.New("修订冲突")
)
