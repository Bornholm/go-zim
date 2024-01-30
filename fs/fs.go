package fs

import (
	"bytes"
	"io/fs"
	iofs "io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/Bornholm/go-zim"
	"github.com/pkg/errors"
)

type FS struct {
	reader *zim.Reader
}

// Open implements fs.FS.
func (fs *FS) Open(name string) (iofs.File, error) {
	switch name {
	case ".":
		return fs.serveDirectory(name)
	case "index.html":
		return fs.serveIndex()
	default:
		return fs.serveZimEntry(name)
	}
}

func (fs *FS) serveIndex() (iofs.File, error) {
	main, err := fs.reader.MainPage()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return fs.serveZimEntry(main.FullURL())
}

func (fs *FS) serveDirectory(name string) (iofs.File, error) {
	zimFile := &Directory{
		File: File{
			fileInfo: &FileInfo{
				isDir:   true,
				modTime: time.Time{},
				mode:    0,
				name:    name,
				size:    0,
			},
			reader: &zim.NoopReadSeekCloser{
				ReadSeeker: bytes.NewReader(nil),
			},
		},
		entries: make([]iofs.DirEntry, 0),
	}

	return zimFile, nil
}

func (fs *FS) serveZimEntry(name string) (iofs.File, error) {
	entry, err := fs.searchEntryFromURL(name)
	if err != nil {
		if errors.Is(err, zim.ErrNotFound) {
			return nil, os.ErrNotExist
		}

		return nil, errors.WithStack(err)
	}

	content, err := entry.Redirect()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	contentReader, err := content.Reader()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	size, err := contentReader.Size()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	filename := filepath.Base(name)

	zimFile := &File{
		fileInfo: &FileInfo{
			isDir:   false,
			modTime: time.Time{},
			mode:    0,
			name:    filename,
			size:    size,
		},
		reader: contentReader,
	}

	return zimFile, nil
}

func (fs *FS) searchEntryFromURL(url string) (zim.Entry, error) {
	entry, err := fs.reader.EntryWithFullURL(url)
	if err != nil && !errors.Is(err, zim.ErrNotFound) {
		return nil, errors.WithStack(err)
	}

	if entry != nil {
		return entry, nil
	}

	contentNamespaces := []zim.Namespace{
		zim.V6NamespaceContent,
		zim.V6NamespaceMetadata,
		zim.V5NamespaceLayout,
		zim.V5NamespaceArticle,
		zim.V5NamespaceImageFile,
		zim.V5NamespaceMetadata,
	}

	for _, ns := range contentNamespaces {
		entry, err := fs.reader.EntryWithURL(ns, url)
		if err != nil && !errors.Is(err, zim.ErrNotFound) {
			return nil, errors.WithStack(err)
		}

		if entry != nil {
			return entry, nil
		}
	}

	iterator := fs.reader.Entries()
	for iterator.Next() {
		current := iterator.Entry()

		if current.FullURL() != url && current.URL() != url {
			continue
		}

		entry = current
		break
	}
	if err := iterator.Err(); err != nil {
		return nil, errors.WithStack(err)
	}

	if entry == nil {
		return nil, errors.WithStack(zim.ErrNotFound)
	}

	return entry, nil
}

func New(reader *zim.Reader) *FS {
	return &FS{reader}
}

var _ fs.FS = &FS{}
