package core

import (
	"bytes"
	"encoding/json"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// JSONStringify is a javascripty way to convert and object to a stringified JSON
// Failes hard with panic, so beware. Mostly good for tests.
func JSONStringify(i interface{}, outBuf *bytes.Buffer) *bytes.Buffer {
	if outBuf == nil {
		outBuf = bytes.NewBuffer(nil)
	}
	err := json.NewEncoder(outBuf).Encode(i)
	if err != nil {
		panic(err)
	}
	return outBuf
}

// JSONParse is a javascripty way to parse JSON. Fails hard with panic, so beware. Mostly for tests.
func JSONParse(i interface{}, inBuf *bytes.Buffer) interface{} {
	err := json.NewDecoder(inBuf).Decode(i)
	if err != nil {
		panic(err)
	}
	return i
}

var hexDigits = "0123456789abcdef"

// StringJSONStringify escapes a string for JSON encoding
func StringJSONStringify(s *string, outBuf *bytes.Buffer) {
	outBuf.WriteByte('"')
	start := 0
	for i := 0; i < len(*s); {
		if b := (*s)[i]; b < utf8.RuneSelf {
			if start < i {
				outBuf.WriteString((*s)[start:i])
			}
			outBuf.WriteByte('\\')
			switch b {
			case '\\', '"':
				outBuf.WriteByte(b)
			case '\n':
				outBuf.WriteByte('n')
			case '\r':
				outBuf.WriteByte('r')
			case '\t':
				outBuf.WriteByte('t')
			default:
				// This encodes bytes < 0x20 except for \t, \n and \r.
				// If escapeHTML is set, it also escapes <, >, and &
				// because they can lead to security holes when
				// user-controlled strings are rendered into JSON
				// and served to some browsers.
				outBuf.WriteString(`u00`)
				outBuf.WriteByte(hexDigits[b>>4])
				outBuf.WriteByte(hexDigits[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString((*s)[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				outBuf.WriteString((*s)[start:i])
			}
			outBuf.WriteString(`\ufffd`)
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				outBuf.WriteString((*s)[start:i])
			}
			outBuf.WriteString(`\u202`)
			outBuf.WriteByte(hexDigits[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len((*s)) {
		outBuf.WriteString((*s)[start:])
	}
	outBuf.WriteByte('"')
}

// StringJSONParse parses out a JSON String
func StringJSONParse(s []byte) (out string, ok bool) {
	// We already know that s[0] == '"'. However, we don't know that the
	// closing quote exists in all cases, such as when the string is nested
	// via the ",string" option.
	if len(s) < 2 || s[len(s)-1] != '"' {
		return
	}
	s = s[1 : len(s)-1]

	// If there are no unusual characters, no unquoting is needed, so return
	// a slice of the original bytes.
	r := 0
	b := make([]byte, len(s)+2*utf8.UTFMax)
	w := copy(b, s[0:r])
	for r < len(s) {
		// Out of room? Can only happen if s is full of
		// malformed UTF-8 and we're replacing each
		// byte with RuneError.
		if w >= len(b)-2*utf8.UTFMax {
			nb := make([]byte, (len(b)+utf8.UTFMax)*2)
			copy(nb, b[0:w])
			b = nb
		}
		switch c := s[r]; {
		case c == '\\':
			r++
			if r >= len(s) {
				return
			}
			switch s[r] {
			default:
				return
			case '"', '\\', '/', '\'':
				b[w] = s[r]
				r++
				w++
			case 'b':
				b[w] = '\b'
				r++
				w++
			case 'f':
				b[w] = '\f'
				r++
				w++
			case 'n':
				b[w] = '\n'
				r++
				w++
			case 'r':
				b[w] = '\r'
				r++
				w++
			case 't':
				b[w] = '\t'
				r++
				w++
			case 'u':
				r--
				rr := getu4(s[r:])
				if rr < 0 {
					return
				}
				r += 6
				if utf16.IsSurrogate(rr) {
					rr1 := getu4(s[r:])
					if dec := utf16.DecodeRune(rr, rr1); dec != unicode.ReplacementChar {
						// A valid pair; consume.
						r += 6
						w += utf8.EncodeRune(b[w:], dec)
						break
					}
					// Invalid surrogate; fall back to replacement rune.
					rr = unicode.ReplacementChar
				}
				w += utf8.EncodeRune(b[w:], rr)
			}

		// Quote, control characters are invalid.
		case c == '"', c < ' ':
			return

		// ASCII
		case c < utf8.RuneSelf:
			b[w] = c
			r++
			w++

		// Coerce to well-formed UTF-8.
		default:
			rr, size := utf8.DecodeRune(s[r:])
			r += size
			w += utf8.EncodeRune(b[w:], rr)
		}
	}
	return string(b[0:w]), true
}

// getu4 decodes \uXXXX from the beginning of s, returning the hex value,
// or it returns -1.
func getu4(s []byte) rune {
	if len(s) < 6 || s[0] != '\\' || s[1] != 'u' {
		return -1
	}
	var r rune
	for _, c := range s[2:6] {
		switch {
		case '0' <= c && c <= '9':
			c = c - '0'
		case 'a' <= c && c <= 'f':
			c = c - 'a' + 10
		case 'A' <= c && c <= 'F':
			c = c - 'A' + 10
		default:
			return -1
		}
		r = r*16 + rune(c)
	}
	return r
}
