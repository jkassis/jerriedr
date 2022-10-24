package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// static authTokenSignature(authToken) {
//     return hash
//       .sha256()
//       .update(
//         `${authToken.type}|${JSON.stringify(authToken.content)}|${authToken.timeMs
//         }|${process.env['APP_AUTH_SIGNING_SECRET']}`
//       )
//       .digest('hex')
//   }

// AuthToken is an authentication token
type AuthToken struct {
	Content struct {
		UserUUID string `json:"userUUID" yaml:"userUUID"`
		Email    string `json:"email,omitempty" yaml:"email,omitempty"`
		Phone    string `json:"phone,omitempty" yaml:"phone,omitempty"`
	}
	Signature string `json:"signature" yaml:"signature"`
	TimeMs    int64  `json:"timeMs" yaml:"timeMs"`
	Type      string `json:"type" yaml:"type"`
}

// SignatureExpected returns the expected signature
func (at *AuthToken) SignatureExpected(secret string) string {
	payload := fmt.Sprintf("%s|%s|%s|%s|%d|%s", at.Type, at.Content.UserUUID, at.Content.Email, at.Content.Phone, at.TimeMs, secret)
	hash := sha256.Sum256([]byte(payload))
	hexHash := hex.EncodeToString(hash[:])
	return hexHash

	// return hash
	// .sha256()
	// .update(
	//   `${authToken.type}|${JSON.stringify(authToken.content)}|${authToken.timeMs
	//   }|${process.env['APP_AUTH_SIGNING_SECRET']}`
	// )
	// .digest('hex')
}
