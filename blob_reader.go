package zim

import "io"

type BlobReader interface {
	io.ReadSeekCloser
	Size() (int64, error)
}
