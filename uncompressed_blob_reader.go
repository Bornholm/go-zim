package zim

import (
	"bytes"
	"sync"

	"github.com/pkg/errors"
)

type UncompressedBlobReader struct {
	reader          *Reader
	blobStartOffset uint64
	blobEndOffset   uint64
	blobSize        int
	readOffset      int

	blob         *bytes.Reader
	loadBlobOnce sync.Once
	loadBlobErr  error
}

// Seek implements BlobReader.
func (r *UncompressedBlobReader) Seek(offset int64, whence int) (int64, error) {
	blob, err := r.loadBlob()
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return blob.Seek(offset, whence)
}

// Size implements BlobReader.
func (r *UncompressedBlobReader) Size() (int64, error) {
	return int64(r.blobEndOffset - r.blobStartOffset), nil
}

// Close implements io.ReadCloser.
func (r *UncompressedBlobReader) Close() error {
	return nil
}

// Read implements io.ReadCloser.
func (r *UncompressedBlobReader) Read(p []byte) (n int, err error) {
	blobData, err := r.loadBlob()
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return blobData.Read(p)
}

func (r *UncompressedBlobReader) loadBlob() (*bytes.Reader, error) {
	r.loadBlobOnce.Do(func() {
		data := make([]byte, r.blobEndOffset-r.blobStartOffset)
		err := r.reader.readRange(int64(r.blobStartOffset), data)
		if err != nil {
			r.loadBlobErr = errors.WithStack(err)
			return
		}

		r.blob = bytes.NewReader(data)
	})
	if r.loadBlobErr != nil {
		return nil, errors.WithStack(r.loadBlobErr)
	}

	return r.blob, nil
}

func NewUncompressedBlobReader(reader *Reader, blobStartOffset, blobEndOffset uint64, blobSize int) *UncompressedBlobReader {
	return &UncompressedBlobReader{
		reader:          reader,
		blobStartOffset: blobStartOffset,
		blobEndOffset:   blobEndOffset,
		blobSize:        blobSize,
		readOffset:      0,
	}
}

var _ BlobReader = &UncompressedBlobReader{}
