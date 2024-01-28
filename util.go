package zim

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
)

// read a little endian uint64
func readUint64(b []byte, order binary.ByteOrder) (uint64, error) {
	var v uint64
	buf := bytes.NewBuffer(b)
	if err := binary.Read(buf, order, &v); err != nil {
		return 0, errors.WithStack(err)
	}

	return v, nil
}

// read a little endian uint32
func readUint32(b []byte, order binary.ByteOrder) (uint32, error) {
	var v uint32
	buf := bytes.NewBuffer(b)
	if err := binary.Read(buf, order, &v); err != nil {
		return 0, errors.WithStack(err)
	}

	return v, nil
}

// read a little endian uint16
func readUint16(b []byte, order binary.ByteOrder) (uint16, error) {
	var v uint16
	buf := bytes.NewBuffer(b)
	if err := binary.Read(buf, order, &v); err != nil {
		return 0, errors.WithStack(err)
	}

	return v, nil
}

// read a little endian uint8
func readUint8(b []byte, order binary.ByteOrder) (uint8, error) {
	var v uint8
	buf := bytes.NewBuffer(b)
	if err := binary.Read(buf, order, &v); err != nil {
		return 0, errors.WithStack(err)
	}

	return v, nil
}
