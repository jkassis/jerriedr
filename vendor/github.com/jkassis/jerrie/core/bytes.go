package core

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// Bytes16Write is a convenient writer for strings that prepends the length
func Bytes16Write(w io.Writer, bs []byte) (err error) {
	length := len(bs)
	if length >= math.MaxUint16 {
		return errors.New("length too long")
	}
	if err := binary.Write(w, binary.BigEndian, uint16(length)); err != nil {
		return err
	}
	_, err = w.Write(bs)
	return err
}

// Bytes16Read is a convenient reader for variable length strings that uses a prepended length
// to know how far to parse
func Bytes16Read(r io.Reader) (val []byte, err error) {
	// Read the length
	var length uint16
	if err = binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	// Read the value
	val = make([]byte, length)
	if lengthRead, err := io.ReadFull(r, val); err != nil {
		return nil, err
	} else if uint16(lengthRead) != length {
		return nil, fmt.Errorf("didn't read correct number of bytes")
	}

	return val, nil
}

// Bytes32Write is a convenient writer for strings that prepends the length
func Bytes32Write(w io.Writer, bs []byte) (err error) {
	length := len(bs)
	if length >= math.MaxUint32 {
		return errors.New("length too long")
	}
	if err := binary.Write(w, binary.BigEndian, uint32(length)); err != nil {
		return err
	}
	_, err = w.Write(bs)
	return err
}

// Bytes32Read is a convenient reader for variable length strings that uses a prepended length
// to know how far to parse
func Bytes32Read(r io.Reader) (val []byte, err error) {
	// Read the length
	var length uint32
	if err = binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	// Read the value
	val = make([]byte, length)
	if lengthRead, err := io.ReadFull(r, val); err != nil {
		return nil, err
	} else if uint32(lengthRead) != length {
		return nil, fmt.Errorf("didn't read correct number of bytes")
	}

	return val, nil
}

// Bytes2Base64 converts a []byte to base64
func Bytes2Base64(s []byte) string {
	return base64.StdEncoding.EncodeToString(s)
}
