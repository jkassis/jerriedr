package core

import (
	"bytes"
	fmt "fmt"
)

// DBString represents a request for a timer
type DBString struct {
	K *DBStringK
	V *DBStringV
}

// NewDBString produces a new DBStringReq in a single line
func NewDBString(key string) *DBString {
	return &DBString{
		K: &DBStringK{Key: key},
		V: &DBStringV{},
	}
}

// MarshalBinary dehydrates to []byte
func (req *DBString) MarshalBinary() ([]byte, error) {
	pk, err := req.K.MarshalBinary()
	if err != nil {
		return nil, err
	}
	ob, err := req.V.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return append(pk, ob...), nil
}

// UnmarshalBinary hydrates from []byte
func (req *DBString) UnmarshalBinary(in []byte) error {
	if err := req.K.UnmarshalBinary(in); err != nil {
		return err
	}

	return req.V.UnmarshalBinary(in[len(req.K.Key)+5:])
}

// DBStringType is the type id for balances
const DBStringType = "str"

func init() {
	DBTypeRegistryGlobal.Register([]byte(DBStringType))
}

// DBStringK is the primary key of a timer
// by using TimeToPlay in the primary key, the timer is sorted in the DB by play time,
// but once the timer is run the first time, a copy is stored in the DB with a new TimeToPlay
// and thus the timer is no longer deleteable by the caller / client.
type DBStringK struct {
	Key string
	DBNodeK
}

func (k *DBStringK) String() string {
	return fmt.Sprintf(`DBStringK{ Key : "%s" }`, k.Key)
}

// MarshalBinary dehydrates to []byte
func (k *DBStringK) MarshalBinary() ([]byte, error) {
	out := bytes.NewBuffer(nil)
	out.Write([]byte(DBStringType))
	String16Write(out, k.Key)
	return out.Bytes(), nil
}

// UnmarshalBinary hydrates from []byte
func (k *DBStringK) UnmarshalBinary(bin []byte) error {
	if key, err := String16Read(bytes.NewBuffer(bin[3:])); err == nil {
		k.Key = key
	} else {
		return err
	}
	return nil
}

// DBStringV represents a callback
type DBStringV struct {
	Value []byte
}

// Indexes returns indexes for the constant
func (v *DBStringV) Indexes() DBIndexKeyGenMap {
	return nil
}

// MarshalBinary dehydrates to []byte
func (v *DBStringV) MarshalBinary() ([]byte, error) {
	outBuf := bytes.NewBuffer(nil)
	Bytes32Write(outBuf, v.Value)
	return outBuf.Bytes(), nil
}

// UnmarshalBinary hydrates from []byte
func (v *DBStringV) UnmarshalBinary(in []byte) (err error) {
	if value, err := Bytes32Read(bytes.NewBuffer(in)); err == nil {
		v.Value = value
	} else {
		return err
	}
	return nil
}
