package core

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	log "github.com/sirupsen/logrus"
)

// JSONEncodedRsaPublicKey : asdf
type JSONEncodedRsaPublicKey rsa.PublicKey

// UnmarshalJSON : asdf
func (key *JSONEncodedRsaPublicKey) UnmarshalJSON(data []byte) error {
	data = bytes.Trim(data, "\"")
	data = bytes.ReplaceAll(data, []byte("\\n"), []byte("\n"))
	log.Debug(string(data))
	block, _ := pem.Decode(data)
	if block == nil {
		return errors.New("failed to parse PEM block containing the key")
	}

	pubKeyGeneric, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}

	switch pubKeyGeneric.(type) {
	case *rsa.PublicKey:
		pubKeyRsa := pubKeyGeneric.(*rsa.PublicKey)
		key.E = pubKeyRsa.E
		key.N = pubKeyRsa.N
	default:
		return errors.New("Key type is not RSA")
	}
	return nil
}

// MarshalBinary dehydrates to []byte
func (key *JSONEncodedRsaPublicKey) MarshalBinary() ([]byte, error) {
	return x509.MarshalPKIXPublicKey((*rsa.PublicKey)(key))
}

// UnmarshalBinary hydrates from []byte
func (key *JSONEncodedRsaPublicKey) UnmarshalBinary(data []byte) error {
	pubKeyGeneric, err := x509.ParsePKIXPublicKey(data)
	if err != nil {
		return err
	}
	pubKeyRsa := pubKeyGeneric.(*rsa.PublicKey)
	key.E = pubKeyRsa.E
	key.N = pubKeyRsa.N
	return nil
}
