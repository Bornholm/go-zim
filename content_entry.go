package zim

import (
	"encoding/binary"

	"github.com/pkg/errors"
)

type zimCompression int

const (
	zimCompressionNoneZeno      zimCompression = 0
	zimCompressionNone          zimCompression = 1
	zimCompressionNoneZLib      zimCompression = 2
	zimCompressionNoneBZip2     zimCompression = 3
	zimCompressionNoneXZ        zimCompression = 4
	zimCompressionNoneZStandard zimCompression = 5
)

type ContentEntry struct {
	*BaseEntry
	mimeType     string
	clusterIndex uint32
	blobIndex    uint32
}

func (e *ContentEntry) Compression() (int, error) {
	clusterHeader, _, _, err := e.readClusterInfo()
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return int((clusterHeader << 4) >> 4), nil
}

func (e *ContentEntry) MimeType() string {
	return e.mimeType
}

func (e *ContentEntry) Reader() (BlobReader, error) {
	clusterHeader, clusterStartOffset, clusterEndOffset, err := e.readClusterInfo()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	compression := (clusterHeader << 4) >> 4
	extended := (clusterHeader<<3)>>7 == 1

	blobSize := 4
	if extended {
		blobSize = 8
	}

	switch compression {

	// Uncompressed blobs
	case uint8(zimCompressionNoneZeno):
		fallthrough
	case uint8(zimCompressionNone):
		startPos := clusterStartOffset + 1
		blobOffset := uint64(e.blobIndex * uint32(blobSize))

		data := make([]byte, 2*blobSize)
		if err := e.reader.readRange(int64(startPos+blobOffset), data); err != nil {
			return nil, errors.WithStack(err)
		}

		var (
			blobStart uint64
			blobEnd   uint64
		)

		if extended {
			blobStart64, err := readUint64(data[0:blobSize], binary.LittleEndian)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			blobStart = blobStart64

			blobEnd64, err := readUint64(data[blobSize:blobSize*2], binary.LittleEndian)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			blobEnd = uint64(blobEnd64)
		} else {
			blobStart32, err := readUint32(data[0:blobSize], binary.LittleEndian)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			blobStart = uint64(blobStart32)

			blobEnd32, err := readUint32(data[blobSize:blobSize*2], binary.LittleEndian)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			blobEnd = uint64(blobEnd32)
		}

		return NewUncompressedBlobReader(e.reader, startPos+blobStart, startPos+blobEnd, blobSize), nil

	// Supported compression algorithms
	case uint8(zimCompressionNoneXZ):
		return NewXZBlobReader(e.reader, clusterStartOffset, clusterEndOffset, e.blobIndex, blobSize), nil

	case uint8(zimCompressionNoneZStandard):
		return NewZStdBlobReader(e.reader, clusterStartOffset, clusterEndOffset, e.blobIndex, blobSize), nil

	// Unsupported compression algorithms
	case uint8(zimCompressionNoneZLib):
		fallthrough
	case uint8(zimCompressionNoneBZip2):
		fallthrough
	default:
		return nil, errors.Wrapf(ErrCompressionAlgorithmNotSupported, "unexpected compression algorithm '%d'", compression)
	}
}

func (e *ContentEntry) Redirect() (*ContentEntry, error) {
	return e, nil
}

func (e *ContentEntry) readClusterInfo() (uint8, uint64, uint64, error) {
	startClusterOffset, clusterEndOffset, err := e.reader.getClusterOffsets(int(e.clusterIndex))
	if err != nil {
		return 0, 0, 0, errors.WithStack(err)
	}

	data := make([]byte, 1)
	if err := e.reader.readRange(int64(startClusterOffset), data); err != nil {
		return 0, 0, 0, errors.WithStack(err)
	}

	clusterHeader := uint8(data[0])

	return clusterHeader, startClusterOffset, clusterEndOffset, nil
}

func (r *Reader) parseContentEntry(offset int64, base *BaseEntry) (*ContentEntry, error) {
	entry := &ContentEntry{
		BaseEntry: base,
	}

	data := make([]byte, 16)
	if err := r.readRange(offset, data); err != nil {
		return nil, errors.WithStack(err)
	}

	mimeTypeIndex, err := readUint16(data[0:2], binary.LittleEndian)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if mimeTypeIndex >= uint16(len(r.mimeTypes)) {
		return nil, errors.Errorf("mime type index '%d' greater than mime types length '%d'", mimeTypeIndex, len(r.mimeTypes))
	}

	entry.mimeType = r.mimeTypes[mimeTypeIndex]

	entry.namespace = Namespace(data[3:4])

	clusterIndex, err := readUint32(data[8:12], binary.LittleEndian)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	entry.clusterIndex = clusterIndex

	blobIndex, err := readUint32(data[12:16], binary.LittleEndian)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	entry.blobIndex = blobIndex

	strs, _, err := r.readStringsAt(offset+16, 2, 1024)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(strs) > 0 {
		entry.url = strs[0]
	}

	if len(strs) > 1 {
		entry.title = strs[1]
	}

	return entry, nil
}
