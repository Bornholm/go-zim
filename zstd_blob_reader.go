package zim

import (
	"bytes"
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/pkg/errors"
)

func NewZStdBlobReader(reader *Reader, clusterStartOffset, clusterEndOffset uint64, blobIndex uint32, blobSize int) *CompressedBlobReader {
	return NewCompressedBlobReader(
		reader,
		func(r io.Reader) (io.ReadSeekCloser, error) {
			decoder, err := zstd.NewReader(r)
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
