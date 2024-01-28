package fs

import (
	"io"
	"io/fs"
	"time"
)

type File struct {
	fileInfo *FileInfo
	reader   io.ReadSeekCloser
}

// Seek implements io.Seeker.
func (f *File) Seek(offset int64, whence int) (int64, error) {
	return f.reader.Seek(offset, whence)
}

// Close implements fs.File.
func (f *File) Close() error {
	return f.reader.Close()
}

// Read implements fs.File.
func (f *File) Read(d []byte) (int, error) {
	return f.reader.Read(d)
}

// Stat implements fs.File.
func (f *File) Stat() (fs.FileInfo, error) {
	return f.fileInfo, nil
}

var (
	_ fs.File   = &File{}
	_ io.Seeker = &File{}
)

type FileInfo struct {
	isDir   bool
	modTime time.Time
	mode    fs.FileMode
	name    string
	size    int64
}

// IsDir implements fs.FileInfo.
func (i *FileInfo) IsDir() bool {
	return i.isDir
}

// ModTime implements fs.FileInfo.
func (i *FileInfo) ModTime() time.Time {
	return i.modTime
}

// Mode implements fs.FileInfo.
func (i *FileInfo) Mode() fs.FileMode {
	return i.mode
}

// Name implements fs.FileInfo.
func (i *FileInfo) Name() string {
	return i.name
}

// Size implements fs.FileInfo.
func (i *FileInfo) Size() int64 {
	return i.size
}

// Sys implements fs.FileInfo.
func (*FileInfo) Sys() any {
	return nil
}

var _ fs.FileInfo = &FileInfo{}

type Directory struct {
	File
	entries []fs.DirEntry
}

// ReadDir implements fs.ReadDirFile.
func (d *Directory) ReadDir(n int) ([]fs.DirEntry, error) {
	return d.entries, nil
}

var _ fs.ReadDirFile = &Directory{}
