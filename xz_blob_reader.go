package zim

import (
	"bytes"
	"io"

	"github.com/pkg/errors"
	"github.com/ulikunitz/xz"
)

func NewXZBlobReader(reader *Reader, clusterStartOffset, clusterEndOffset uint64, blobIndex uint32, blobSize int) *CompressedBlobReader {
	return NewCompressedBlobReader(
		reader,
		func(r io.Reader) (io.ReadSeekCloser, error) {
			decoder, err := xz.NewReader(r)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			buff, err := io.ReadAll(decoder)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			return &NoopReadSeekCloser{bytes.NewReader(buff)}, nil
		},
		clusterStartOffset,
		clusterEndOffset,
		blobIndex,
		blobSize,
	)
}
