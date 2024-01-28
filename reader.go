package zim

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	"gitlab.com/wpetit/goweb/logger"
)

const zimFormatMagicNumber uint32 = 0x44D495A
const nullByte = '\x00'
const zimRedirect = 0xffff

type Reader struct {
	majorVersion  uint16
	minorVersion  uint16
	uuid          string
	entryCount    uint32
	clusterCount  uint32
	urlPtrPos     uint64
	titlePtrPos   uint64
	clusterPtrPos uint64
	mimeListPos   uint64
	mainPage      uint32
	layoutPage    uint32
	checksumPos   uint64

	mimeTypes    []string
	urlIndex     []uint64
	clusterIndex []uint64

	cache *lru.Cache[string, Entry]
	urls  map[string]int

	rangeReader RangeReadCloser
}

func (r *Reader) Version() (majorVersion, minorVersion uint16) {
	return r.majorVersion, r.minorVersion
}

func (r *Reader) EntryCount() uint32 {
	return r.entryCount
}

func (r *Reader) ClusterCount() uint32 {
	return r.clusterCount
}

func (r *Reader) UUID() string {
	return r.uuid
}

func (r *Reader) Close() error {
	if err := r.rangeReader.Close(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (r *Reader) MainPage() (Entry, error) {
	if r.mainPage == 0xffffffff {
		return nil, errors.WithStack(ErrNotFound)
	}

	entry, err := r.EntryAt(int(r.mainPage))
	if err != nil {
		return nil, errors.WithStack(ErrNotFound)
	}

	return entry, nil
}

func (r *Reader) Entries() *EntryIterator {
	return &EntryIterator{
		reader: r,
	}
}

func (r *Reader) EntryAt(idx int) (Entry, error) {
	if idx >= len(r.urlIndex) || idx < 0 {
		return nil, errors.Wrapf(ErrInvalidIndex, "index '%d' out of bounds", idx)
	}

	entryPtr := r.urlIndex[idx]

	entry, err := r.parseEntryAt(int64(entryPtr))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	r.cacheEntry(entryPtr, entry)

	return entry, nil
}

func (r *Reader) EntryWithFullURL(url string) (Entry, error) {
	urlNum, exists := r.urls[url]
	if !exists {
		return nil, errors.WithStack(ErrNotFound)
	}

	entry, err := r.EntryAt(urlNum)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return entry, nil
}

func (r *Reader) EntryWithURL(ns Namespace, url string) (Entry, error) {
	fullURL := toFullURL(ns, url)

	entry, err := r.EntryWithFullURL(fullURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return entry, nil
}

func (r *Reader) EntryWithTitle(ns Namespace, title string) (Entry, error) {
	entry, found := r.getEntryByTitleFromCache(ns, title)
	if found {
		logger.Debug(context.Background(), "found entry with title from cache", logger.F("entry", entry.FullURL()))
		return entry, nil
	}

	iterator := r.Entries()

	for iterator.Next() {
		entry := iterator.Entry()

		if entry.Title() == title && entry.Namespace() == ns {
			return entry, nil
		}
	}
	if err := iterator.Err(); err != nil {
		return nil, errors.WithStack(err)
	}

	return nil, errors.WithStack(ErrNotFound)
}

func (r *Reader) getURLCacheKey(fullURL string) string {
	return "url:" + fullURL
}

func (r *Reader) getTitleCacheKey(ns Namespace, title string) string {
	return fmt.Sprintf("title:%s/%s", ns, title)
}

func (r *Reader) cacheEntry(offset uint64, entry Entry) {
	urlKey := r.getURLCacheKey(entry.FullURL())
	titleKey := r.getTitleCacheKey(entry.Namespace(), entry.Title())

	_, urlFound := r.cache.Peek(urlKey)
	_, titleFound := r.cache.Peek(titleKey)

	if urlFound && titleFound {
		return
	}

	r.cache.Add(urlKey, entry)
	r.cache.Add(titleKey, entry)
}

func (r *Reader) getEntryByTitleFromCache(namespace Namespace, title string) (Entry, bool) {
	key := r.getTitleCacheKey(namespace, title)
	return r.cache.Get(key)
}

func (r *Reader) parse() error {
	if err := r.parseHeader(); err != nil {
		return errors.WithStack(err)
	}

	if err := r.parseMimeTypes(); err != nil {
		return errors.WithStack(err)
	}

	if err := r.parseURLIndex(); err != nil {
		return errors.WithStack(err)
	}

	if err := r.parseClusterIndex(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (r *Reader) parseHeader() error {
	header := make([]byte, 80)
	if err := r.readRange(0, header); err != nil {
		return errors.WithStack(err)
	}

	magicNumber, err := readUint32(header[0:4], binary.LittleEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	if magicNumber != zimFormatMagicNumber {
		return errors.Errorf("invalid zim magic number '%d'", magicNumber)
	}

	majorVersion, err := readUint16(header[4:6], binary.LittleEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	r.majorVersion = majorVersion

	minorVersion, err := readUint16(header[6:8], binary.LittleEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	r.minorVersion = minorVersion

	if err := r.parseUUID(header[8:16]); err != nil {
		return errors.WithStack(err)
	}

	entryCount, err := readUint32(header[24:28], binary.LittleEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	r.entryCount = entryCount

	clusterCount, err := readUint32(header[28:32], binary.LittleEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	r.clusterCount = clusterCount

	urlPtrPos, err := readUint64(header[32:40], binary.LittleEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	r.urlPtrPos = urlPtrPos

	titlePtrPos, err := readUint64(header[40:48], binary.LittleEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	r.titlePtrPos = titlePtrPos

	clusterPtrPos, err := readUint64(header[48:56], binary.LittleEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	r.clusterPtrPos = clusterPtrPos

	mimeListPos, err := readUint64(header[56:64], binary.LittleEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	r.mimeListPos = mimeListPos

	mainPage, err := readUint32(header[64:68], binary.LittleEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	r.mainPage = mainPage

	layoutPage, err := readUint32(header[68:72], binary.LittleEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	r.layoutPage = layoutPage

	checksumPos, err := readUint64(header[72:80], binary.LittleEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	r.checksumPos = checksumPos

	return nil
}

func (r *Reader) parseUUID(data []byte) error {
	parts := make([]string, 0, 5)

	val32, err := readUint32(data[0:4], binary.BigEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	parts = append(parts, fmt.Sprintf("%08x", val32))

	val16, err := readUint16(data[4:6], binary.BigEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	parts = append(parts, fmt.Sprintf("%04x", val16))

	val16, err = readUint16(data[6:8], binary.BigEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	parts = append(parts, fmt.Sprintf("%04x", val16))

	val16, err = readUint16(data[8:10], binary.BigEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	parts = append(parts, fmt.Sprintf("%04x", val16))

	val32, err = readUint32(data[10:14], binary.BigEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	val16, err = readUint16(data[14:16], binary.BigEndian)
	if err != nil {
		return errors.WithStack(err)
	}

	parts = append(parts, fmt.Sprintf("%x%x", val32, val16))

	r.uuid = strings.Join(parts, "-")

	return nil
}

func (r *Reader) parseMimeTypes() error {
	mimeTypes := make([]string, 0)
	offset := int64(r.mimeListPos)
	read := int64(0)
	var err error
	var found []string
	for {
		found, read, err = r.readStringsAt(offset+read, 64, 1024)
		if err != nil && !errors.Is(err, io.EOF) {
			return errors.WithStack(err)
		}

		if len(found) == 0 || found[0] == "" {
			break
		}

		mimeTypes = append(mimeTypes, found...)
	}

	r.mimeTypes = mimeTypes

	return nil
}

func (r *Reader) parseURLIndex() error {
	urlIndex, err := r.parsePointerIndex(int64(r.urlPtrPos), int64(r.entryCount))
	if err != nil {
		return errors.WithStack(err)
	}

	r.urlIndex = urlIndex

	return nil
}

func (r *Reader) parseClusterIndex() error {
	clusterIndex, err := r.parsePointerIndex(int64(r.clusterPtrPos), int64(r.clusterCount+1))
	if err != nil {
		return errors.WithStack(err)
	}

	r.clusterIndex = clusterIndex

	return nil
}

func (r *Reader) parseEntryAt(offset int64) (Entry, error) {
	base, err := r.parseBaseEntry(offset)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var entry Entry

	if base.mimeTypeIndex == zimRedirect {
		entry, err = r.parseRedirectEntry(offset, base)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	} else {
		entry, err = r.parseContentEntry(offset, base)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return entry, nil
}

func (r *Reader) parsePointerIndex(startAddr int64, count int64) ([]uint64, error) {
	index := make([]uint64, count)

	data := make([]byte, count*8)
	if err := r.readRange(startAddr, data); err != nil {
		return nil, errors.WithStack(err)
	}

	for i := int64(0); i < count; i++ {
		offset := i * 8
		ptr, err := readUint64(data[offset:offset+8], binary.LittleEndian)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		index[i] = ptr
	}

	return index, nil
}

func (r *Reader) getClusterOffsets(clusterNum int) (uint64, uint64, error) {
	if clusterNum > len(r.clusterIndex)-1 || clusterNum < 0 {
		return 0, 0, errors.Wrapf(ErrInvalidIndex, "index '%d' out of bounds", clusterNum)
	}

	return r.clusterIndex[clusterNum], r.clusterIndex[clusterNum+1] - 1, nil
}

func (r *Reader) preload() error {
	r.urls = make(map[string]int, r.entryCount)

	iterator := r.Entries()
	for iterator.Next() {
		entry := iterator.Entry()
		r.urls[entry.FullURL()] = iterator.Index()
	}
	if err := iterator.Err(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (r *Reader) readRange(offset int64, v []byte) error {
	read, err := r.rangeReader.ReadAt(v, offset)
	if err != nil {
		return errors.WithStack(err)
	}

	if read != len(v) {
		return io.EOF
	}

	return nil
}

func (r *Reader) readStringsAt(offset int64, count int, bufferSize int) ([]string, int64, error) {
	var sb strings.Builder
	read := int64(0)

	values := make([]string, 0, count)
	wasNullByte := false

	for {
		data := make([]byte, bufferSize)
		err := r.readRange(offset+read, data)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, read, errors.WithStack(err)
		}

		for idx := 0; idx < len(data); idx++ {
			d := data[idx]
			if err := sb.WriteByte(d); err != nil {
				return nil, read, errors.WithStack(err)
			}

			read++

			if d == nullByte {
				if wasNullByte {
					return values, read, nil
				}

				wasNullByte = true

				str := strings.TrimRight(sb.String(), "\x00")
				values = append(values, str)

				if len(values) == count || errors.Is(err, io.EOF) {
					return values, read, nil
				}

				sb.Reset()
			} else {
				wasNullByte = false
			}
		}
	}
}

type RangeReadCloser interface {
	io.Closer
	ReadAt(data []byte, offset int64) (n int, err error)
}

func NewReader(rangeReader RangeReadCloser, funcs ...OptionFunc) (*Reader, error) {
	opts := NewOptions(funcs...)

	cache, err := lru.New[string, Entry](opts.CacheSize)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	reader := &Reader{
		rangeReader: rangeReader,
		cache:       cache,
	}

	if err := reader.parse(); err != nil {
		return nil, errors.WithStack(err)
	}

	if err := reader.preload(); err != nil {
		return nil, errors.WithStack(err)
	}

	return reader, nil
}

func Open(path string, funcs ...OptionFunc) (*Reader, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	reader, err := NewReader(file, funcs...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return reader, nil
}
