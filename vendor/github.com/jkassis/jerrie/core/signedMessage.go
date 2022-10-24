package core

import (
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/ed25519"
)

// Signer encapsulates a user id, PublicKey, and Signature
type Signer struct {
	UUID      uuid.UUID
	PublicKey []byte `json:"publicKey"`
	Signature []byte `json:"signature"`
}

// SignedMessage encapsulates a message and signatures
type SignedMessage struct {
	Body    string
	Signers []Signer
}

// // Validate checks that all signatures on a message are cryptographically valide
// func (signedMessage *SignedMessage) Validate() error {
// 	signedMessageHash := sha256.Sum256([]byte(signedMessage.Body))
// 	for _, signer := range signedMessage.Signers {
// 		signerBytes, _ := hex.DecodeString(signer.Signature)
// 		err := rsa.VerifyKCS1v15((*rsa.PublicKey)(&signer.PublicKey), crypto.SHA256, signedMessageHash[:], signerBytes)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// Validate checks that all signatures on a message are cryptographically valide
func (signedMessage *SignedMessage) Validate() error {
	for _, signer := range signedMessage.Signers {
		valid := ed25519.Verify(ed25519.PublicKey(signer.PublicKey), []byte(signedMessage.Body), signer.Signature)
		if !valid {
			return errors.New("invalid signature")
		}
	}
	return nil
}

// UnmarshalJSON extracts a signer object from JSON
func (u *Signer) UnmarshalJSON(data []byte) error {
	parts := make(map[string]interface{})
	json.Unmarshal(data, &parts)
	UUID, err := uuid.Parse(parts["UUID"].(string))
	if err != nil {
		return err
	}
	u.UUID = UUID
	PublicKey, err := base64.StdEncoding.DecodeString(parts["publicKey"].(string))
	u.PublicKey = PublicKey
	Signature, err := base64.StdEncoding.DecodeString(parts["signature"].(string))
	u.Signature = Signature
	return nil
}
