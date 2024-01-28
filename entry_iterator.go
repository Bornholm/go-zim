package zim

import "github.com/pkg/errors"

type EntryIterator struct {
	index  int
	entry  Entry
	err    error
	reader *Reader
}

func (it *EntryIterator) Next() bool {
	if it.err != nil {
		return false
	}

	entryCount := it.reader.EntryCount()

	if it.index >= int(entryCount-1) {
		return false
	}

	entry, err := it.reader.EntryAt(it.index)
	if err != nil {
		it.err = errors.WithStack(err)

		return false
	}

	it.entry = entry
	it.index++

	return true
}

func (it *EntryIterator) Err() error {
	return it.err
}

func (it *EntryIterator) Index() int {
	return it.index - 1
}

func (it *EntryIterator) Entry() Entry {
	return it.entry
}
