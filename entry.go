package zim

import (
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
)

type Entry interface {
	Redirect() (*ContentEntry, error)
	Namespace() Namespace
	URL() string
	FullURL() string
	Title() string
}

type BaseEntry struct {
	mimeTypeIndex uint16
	namespace     Namespace
	url           string
	title         string
	reader        *Reader
}

func (e *BaseEntry) Namespace() Namespace {
	return e.namespace
}

func (e *BaseEntry) Title() string {
	if e.title == "" {
		return e.url
	}

	return e.title
}

func (e *BaseEntry) URL() string {
	return e.url
}

func (e *BaseEntry) FullURL() string {
	return toFullURL(e.Namespace(), e.URL())
}

func (r *Reader) parseBaseEntry(offset int64) (*BaseEntry, error) {
	entry := &BaseEntry{
		reader: r,
	}

	data := make([]byte, 3)
	if err := r.readRange(offset, data); err != nil {
		return nil, errors.WithStack(err)
	}

	mimeTypeIndex, err := readUint16(data[0:2], binary.LittleEndian)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	entry.mimeTypeIndex = mimeTypeIndex
	entry.namespace = Namespace(data[2])

	return entry, nil
}

type RedirectEntry struct {
	*BaseEntry
	redirectIndex uint32
}

func (e *RedirectEntry) Redirect() (*ContentEntry, error) {
	if e.redirectIndex >= uint32(len(e.reader.urlIndex)) {
		return nil, errors.Wrapf(ErrInvalidIndex, "entry index '%d' out of bounds", e.redirectIndex)
	}

	entryPtr := e.reader.urlIndex[e.redirectIndex]
	entry, err := e.reader.parseEntryAt(int64(entryPtr))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	entry, err = entry.Redirect()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	contentEntry, ok := entry.(*ContentEntry)
	if !ok {
		return nil, errors.WithStack(ErrInvalidRedirect)
	}

	return contentEntry, nil
}

func (r *Reader) parseRedirectEntry(offset int64, base *BaseEntry) (*RedirectEntry, error) {
	entry := &RedirectEntry{
		BaseEntry: base,
	}

	data := make([]byte, 4)
	if err := r.readRange(offset+8, data); err != nil {
		return nil, errors.WithStack(err)
	}

	redirectIndex, err := readUint32(data, binary.LittleEndian)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	entry.redirectIndex = redirectIndex

	strs, _, err := r.readStringsAt(offset+12, 2, 1024)
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

func toFullURL(ns Namespace, url string) string {
	if ns == "\x00" {
		return url
	}

	return fmt.Sprintf("%s/%s", ns, url)
}
