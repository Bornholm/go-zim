package zim

import "errors"

var (
	ErrInvalidIndex                     = errors.New("invalid index")
	ErrNotFound                         = errors.New("not found")
	ErrInvalidRedirect                  = errors.New("invalid redirect")
	ErrCompressionAlgorithmNotSupported = errors.New("compression algorithm not supported")
)
