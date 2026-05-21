package core

import (
	"crypto/rand"
	"time"
)

const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NewULID generates a new ULID string representing the current time.
// It uses crypto/rand for entropy.
func NewULID() string {
	return NewULIDAt(time.Now())
}

// NewULIDAt generates a new ULID string representing the given time.
func NewULIDAt(t time.Time) string {
	ms := t.UnixMilli()

	// 6 bytes of timestamp (48 bits)
	var ts [6]byte
	ts[0] = byte(ms >> 40)
	ts[1] = byte(ms >> 32)
	ts[2] = byte(ms >> 24)
	ts[3] = byte(ms >> 16)
	ts[4] = byte(ms >> 8)
	ts[5] = byte(ms)

	// 10 bytes of randomness (80 bits)
	var entropy [10]byte
	_, _ = rand.Read(entropy[:])

	var dst [26]byte

	// Encode timestamp
	dst[0] = alphabet[(ts[0]&224)>>5]
	dst[1] = alphabet[ts[0]&31]
	dst[2] = alphabet[(ts[1]&248)>>3]
	dst[3] = alphabet[((ts[1]&7)<<2)|((ts[2]&192)>>6)]
	dst[4] = alphabet[(ts[2]&62)>>1]
	dst[5] = alphabet[((ts[2]&1)<<4)|((ts[3]&240)>>4)]
	dst[6] = alphabet[((ts[3]&15)<<1)|((ts[4]&128)>>7)]
	dst[7] = alphabet[(ts[4]&124)>>2]
	dst[8] = alphabet[((ts[4]&3)<<3)|((ts[5]&224)>>5)]
	dst[9] = alphabet[ts[5]&31]

	// Encode entropy
	r0, r1, r2, r3, r4, r5, r6, r7, r8, r9 := entropy[0], entropy[1], entropy[2], entropy[3], entropy[4], entropy[5], entropy[6], entropy[7], entropy[8], entropy[9]

	dst[10] = alphabet[(r0&248)>>3]
	dst[11] = alphabet[((r0&7)<<2)|((r1&192)>>6)]
	dst[12] = alphabet[(r1&62)>>1]
	dst[13] = alphabet[((r1&1)<<4)|((r2&240)>>4)]
	dst[14] = alphabet[((r2&15)<<1)|((r3&128)>>7)]
	dst[15] = alphabet[(r3&124)>>2]
	dst[16] = alphabet[((r3&3)<<3)|((r4&224)>>5)]
	dst[17] = alphabet[r4&31]

	dst[18] = alphabet[(r5&248)>>3]
	dst[19] = alphabet[((r5&7)<<2)|((r6&192)>>6)]
	dst[20] = alphabet[(r6&62)>>1]
	dst[21] = alphabet[((r6&1)<<4)|((r7&240)>>4)]
	dst[22] = alphabet[((r7&15)<<1)|((r8&128)>>7)]
	dst[23] = alphabet[(r8&124)>>2]
	dst[24] = alphabet[((r8&3)<<3)|((r9&224)>>5)]
	dst[25] = alphabet[r9&31]

	return string(dst[:])
}

// ULIDTime extracts the timestamp from a 26-character ULID string.
func ULIDTime(id string) time.Time {
	if len(id) < 10 {
		return time.Time{}
	}
	var val int64
	for i := 0; i < 10; i++ {
		v := decodeChar(id[i])
		val = (val << 5) | v
	}
	return time.UnixMilli(val)
}

func decodeChar(c byte) int64 {
	switch {
	case c >= '0' && c <= '9':
		return int64(c - '0')
	case c >= 'A' && c <= 'H':
		return int64(c - 'A' + 10)
	case c == 'J' || c == 'K':
		return int64(c - 'J' + 18)
	case c == 'M' || c == 'N':
		return int64(c - 'M' + 20)
	case c >= 'P' && c <= 'T':
		return int64(c - 'P' + 22)
	case c >= 'V' && c <= 'Z':
		return int64(c - 'V' + 27)
	case c >= 'a' && c <= 'h':
		return int64(c - 'a' + 10)
	case c == 'j' || c == 'k':
		return int64(c - 'j' + 18)
	case c == 'm' || c == 'n':
		return int64(c - 'm' + 20)
	case c >= 'p' && c <= 't':
		return int64(c - 'p' + 22)
	case c >= 'v' && c <= 'z':
		return int64(c - 'v' + 27)
	default:
		return 0
	}
}
