package core

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// String16Write is a convenient writer for strings that prepends the length
func String16Write(w io.Writer, str string) (err error) {
	length := len(str)
	if length >= math.MaxUint16 {
		return errors.New("length too long")
	}
	if err = binary.Write(w, binary.BigEndian, uint16(length)); err != nil {
		return err
	}
	_, err = w.Write([]byte(str))
	return err
}

// String16Read is a convenient reader for variable length strings that uses a prepended length
// to know how far to parse
func String16Read(r io.Reader) (val string, err error) {
	// Read the length
	var length uint16
	if err = binary.Read(r, binary.BigEndian, &length); err != nil {
		return "", err
	}

	// Read the value
	valBytes := make([]byte, length)
	if _, err := io.ReadFull(r, valBytes); err != nil {
		return "", err
	}

	return string(valBytes), nil
}

// String32Write is a convenient writer for strings that prepends the length
func String32Write(w io.Writer, str string) (err error) {
	length := len(str)
	if length >= math.MaxUint32 {
		return errors.New("length too long")
	}
	if err = binary.Write(w, binary.BigEndian, uint32(length)); err != nil {
		return err
	}
	_, err = w.Write([]byte(str))
	return err
}

// String32Read is a convenient reader for variable length strings that uses a prepended length
// to know how far to parse
func String32Read(r io.Reader) (val string, err error) {
	// Read the length
	var length uint32
	if err = binary.Read(r, binary.BigEndian, &length); err != nil {
		return "", err
	}

	// Read the value
	valBytes := make([]byte, length)
	if _, err := io.ReadFull(r, valBytes); err != nil {
		return "", err
	}

	return string(valBytes), nil
}

// StringInSlice does a linear search on list and returns true if the string is found
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// String2Base64 converts a string to base64
func String2Base64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
