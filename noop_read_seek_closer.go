package zim

import "io"

type NoopReadSeekCloser struct {
	io.ReadSeeker
}

// Close implements io.Closer.
func (*NoopReadSeekCloser) Close() error {
	return nil
}

var _ io.Closer = &NoopReadSeekCloser{}
