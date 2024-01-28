package zim

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"sync"

	"github.com/pkg/errors"
)

type CompressedBlobReader struct {
	reader         *Reader
	decoderFactory BlobDecoderFactory

	clusterStartOffset uint64
	clusterEndOffset   uint64
	blobIndex          uint32
	blobSize           int
	readOffset         uint64

	loadCluster    sync.Once
	loadClusterErr error

	data   *bytes.Reader
	closed bool
}

// Seek implements BlobReader.
func (r *CompressedBlobReader) Seek(offset int64, whence int) (int64, error) {
	if err := r.loadClusterData(); err != nil {
		return 0, errors.WithStack(err)
	}

	return r.data.Seek(offset, whence)
}

// Size implements BlobReader.
func (r *CompressedBlobReader) Size() (int64, error) {
	if err := r.loadClusterData(); err != nil {
		return 0, errors.WithStack(err)
	}

	return r.data.Size(), nil
}

// Close implements io.ReadCloser.
func (r *CompressedBlobReader) Close() error {
	return nil
}

// Read implements io.ReadCloser.
func (r *CompressedBlobReader) Read(p []byte) (int, error) {
	if err := r.loadClusterData(); err != nil {
		return 0, errors.WithStack(err)
	}

	return r.data.Read(p)
}

func (r *CompressedBlobReader) loadClusterData() error {
	if r.closed {
		return errors.WithStack(os.ErrClosed)
	}

	r.loadCluster.Do(func() {
		compressedData := make([]byte, r.clusterEndOffset-r.clusterStartOffset)
		if err := r.reader.readRange(int64(r.clusterStartOffset+1), compressedData); err != nil {
			r.loadClusterErr = errors.WithStack(err)
			return
		}

		blobBuffer := bytes.NewBuffer(compressedData)

		decoder, err := r.decoderFactory(blobBuffer)
		if err != nil {
			r.loadClusterErr = errors.WithStack(err)
			return
		}

		defer decoder.Close()

		uncompressedData, err := io.ReadAll(decoder)
		if err != nil {
			r.loadClusterErr = errors.WithStack(err)
			return
		}

		var (
			blobStart uint64
			blobEnd   uint64
		)

		if r.blobSize == 8 {
			blobStart64, err := readUint64(uncompressedData[r.blobIndex*uint32(r.blobSize):r.blobIndex*uint32(r.blobSize)+uint32(r.blobSize)], binary.LittleEndian)
			if err != nil {
				r.loadClusterErr = errors.WithStack(err)
				return
			}

			blobStart = blobStart64

			blobEnd64, err := readUint64(uncompressedData[r.blobIndex*uint32(r.blobSize)+uint32(r.blobSize):r.blobIndex*uint32(r.blobSize)+uint32(r.blobSize)+uint32(r.blobSize)], binary.LittleEndian)
			if err != nil {
				r.loadClusterErr = errors.WithStack(err)
				return
			}

			blobEnd = blobEnd64
		} else {
			blobStart32, err := readUint32(uncompressedData[r.blobIndex*uint32(r.blobSize):r.blobIndex*uint32(r.blobSize)+uint32(r.blobSize)], binary.LittleEndian)
			if err != nil {
				r.loadClusterErr = errors.WithStack(err)
				return
			}

			blobStart = uint64(blobStart32)

			blobEnd32, err := readUint32(uncompressedData[r.blobIndex*uint32(r.blobSize)+uint32(r.blobSize):r.blobIndex*uint32(r.blobSize)+uint32(r.blobSize)+uint32(r.blobSize)], binary.LittleEndian)
			if err != nil {
				r.loadClusterErr = errors.WithStack(err)
				return
			}

			blobEnd = uint64(blobEnd32)
		}

		data := make([]byte, blobEnd-blobStart)
		copy(data, uncompressedData[blobStart:blobEnd])

		r.data = bytes.NewReader(data)
	})
	if r.loadClusterErr != nil {
		return errors.WithStack(r.loadClusterErr)
	}

	return nil
}

type BlobDecoderFactory func(io.Reader) (io.ReadSeekCloser, error)

func NewCompressedBlobReader(reader *Reader, decoderFactory BlobDecoderFactory, clusterStartOffset, clusterEndOffset uint64, blobIndex uint32, blobSize int) *CompressedBlobReader {
	return &CompressedBlobReader{
		reader:             reader,
		decoderFactory:     decoderFactory,
		clusterStartOffset: clusterStartOffset,
		clusterEndOffset:   clusterEndOffset,
		blobIndex:          blobIndex,
		blobSize:           blobSize,
		readOffset:         0,
	}
}

var (
	_ BlobReader = &CompressedBlobReader{}
)
