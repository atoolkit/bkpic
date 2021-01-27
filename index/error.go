package index

import "errors"

var (
	ErrNotDirectory  = errors.New("file is not directory")
	ErrInvalidFile   = errors.New("invalid file")
	ErrInvalidMedium = errors.New("invalid medium")
)
