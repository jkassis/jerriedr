package core

import (
	"bytes"
	"fmt"
)

//
// GraphDB structures
//
var (
	// DBKClusterDefault is a blank primary key
	// us-west-2
	DBKClusterDefault = []byte("usw2")
)

// DBNodeK Represents a primary key of a node in a graph structure
// It gives you default implementations for ClusterKeySet and ClusterKeyGet
type DBNodeK struct{}

// ClusterKeySet is a setter
func (k *DBNodeK) ClusterKeySet(clusterKey []byte) {
}

// ClusterKeyGet is a getter
func (k *DBNodeK) ClusterKeyGet() []byte {
	return DBKClusterDefault
}

// IsADBK  always return true
func (k *DBNodeK) IsADBK() bool {
	return true
}

var dbEdgeType = []byte("EDG")

func init() {
	DBTypeRegistryGlobal.Register(dbEdgeType)
}

// DBEdgeK represents the primary key for an edge in a graph db structure
type DBEdgeK struct {
	DBNodeK
	SubType string
	HeadK   DBK
	TailK   DBK
}

func (k *DBEdgeK) String() string {
	if k.TailK == nil {
		return fmt.Sprintf(`DBEdgeK{ SubType: "%s", HeadK: %s, TailK: nil }`, k.SubType, k.HeadK.String())
	}
	return fmt.Sprintf(`DBEdgeK{ SubType: "%s", HeadK: %s, TailK: %s }`, k.SubType, k.HeadK.String(), k.TailK.String())
}

// MarshalBinary dehydrates to []byte
func (k *DBEdgeK) MarshalBinary() ([]byte, error) {
	var err error
	var headK, tailK []byte

	// Type
	outBuf := bytes.NewBuffer(nil)
	if _, err = outBuf.Write(dbEdgeType); err != nil {
		return nil, err
	}

	// SubType
	err = String16Write(outBuf, k.SubType)
	if err != nil {
		return nil, err
	}

	// HeadK
	headK, err = k.HeadK.MarshalBinary()
	if err != nil {
		return nil, err
	}
	if err = Bytes16Write(outBuf, headK); err != nil {
		return nil, err
	}

	// TailK
	if k.TailK != nil {
		tailK, err = k.TailK.MarshalBinary()
		if err != nil {
			return nil, err
		}
		err = Bytes16Write(outBuf, tailK)
		if err != nil {
			return nil, err
		}
	}
	return outBuf.Bytes(), nil
}

// UnmarshalBinary hydrates from []byte
func (k *DBEdgeK) UnmarshalBinary(data []byte) (err error) {
	// Type
	data = data[len(dbEdgeType):]

	// SubType
	inBuf := bytes.NewBuffer(data)
	subType, err := String16Read(inBuf)
	k.SubType = subType

	// HeadK
	headKBytes, err := Bytes16Read(inBuf)
	if err != nil {
		return err
	}
	if err = k.HeadK.UnmarshalBinary(headKBytes); err != nil {
		return err
	}

	// TailK
	tailKBytes, err := Bytes16Read(inBuf)
	if err != nil {
		return err
	}
	return k.TailK.UnmarshalBinary(tailKBytes)
}

// DBEdge is an edge in a graph db structure
type DBEdge struct {
	K *DBEdgeK
}

// Indexes produces a map of indexes
func (dbEdge *DBEdge) Indexes() DBIndexKeyGenMap {
	return make(map[string]DBIndexKeyGen)
}

// MarshalBinary dehydrates to []byte
func (dbEdge *DBEdge) MarshalBinary() ([]byte, error) {
	return make([]byte, 0), nil
}

// UnmarshalBinary hydrates from []byte
func (dbEdge *DBEdge) UnmarshalBinary(data []byte) error {
	return nil
}
