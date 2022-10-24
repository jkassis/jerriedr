package core

import (
	"bytes"
	"encoding/binary"
	fmt "fmt"
)

// DBInt64 represents a request for a timer
type DBInt64 struct {
	K *DBInt64K
	V *DBInt64V
}

// NewDBInt64 produces a new DBInt64Req in a single line
func NewDBInt64(key string) *DBInt64 {
	return &DBInt64{
		K: &DBInt64K{Key: key},
		V: &DBInt64V{},
	}
}

// MarshalBinary dehydrates to []byte
func (req *DBInt64) MarshalBinary() ([]byte, error) {
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
func (req *DBInt64) UnmarshalBinary(in []byte) error {
	if err := req.K.UnmarshalBinary(in); err != nil {
		return err
	}

	return req.V.UnmarshalBinary(in[len(req.K.Key)+5:])
}

// DBInt64Type is the type id for balances
const DBInt64Type = "i64"

func init() {
	DBTypeRegistryGlobal.Register([]byte(DBInt64Type))
}

// DBInt64K is the primary key of a timer
// by using TimeToPlay in the primary key, the timer is sorted in the DB by play time,
// but once the timer is run the first time, a copy is stored in the DB with a new TimeToPlay
// and thus the timer is no longer deleteable by the caller / client.
type DBInt64K struct {
	Key string
	DBNodeK
}

func (k *DBInt64K) String() string {
	return fmt.Sprintf(`DBInt64K{ Key : "%s" }`, k.Key)
}

// MarshalBinary dehydrates to []byte
func (k *DBInt64K) MarshalBinary() ([]byte, error) {
	out := bytes.NewBuffer(nil)
	out.Write([]byte(DBInt64Type))
	String16Write(out, k.Key)
	return out.Bytes(), nil
}

// UnmarshalBinary hydrates from []byte
func (k *DBInt64K) UnmarshalBinary(bin []byte) error {
	if key, err := String16Read(bytes.NewBuffer(bin[3:])); err == nil {
		k.Key = key
	} else {
		return err
	}
	return nil
}

// DBInt64V represents a callback
type DBInt64V struct {
	Value uint64
}

// Indexes returns indexes for the constant
func (v *DBInt64V) Indexes() DBIndexKeyGenMap {
	return nil
}

// MarshalBinary dehydrates to []byte
func (v *DBInt64V) MarshalBinary() ([]byte, error) {
	w := bytes.NewBuffer(nil)
	if err := binary.Write(w, binary.BigEndian, &v.Value); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// UnmarshalBinary hydrates from []byte
func (v *DBInt64V) UnmarshalBinary(in []byte) (err error) {
	r := bytes.NewBuffer(in)
	return binary.Read(r, binary.BigEndian, &v.Value)
}
