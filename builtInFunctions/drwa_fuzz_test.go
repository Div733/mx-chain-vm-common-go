package builtInFunctions

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func FuzzDecodeDRWABinaryHolderMirror(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 8))
	f.Add(make([]byte, drwaBinaryHolderPayloadMinSize))

	valid := make([]byte, 0, 96)
	valid = append(valid, make([]byte, 8)...)
	valid = appendLenPrefixed(valid, []byte("approved"))
	valid = appendLenPrefixed(valid, []byte("clear"))
	valid = appendLenPrefixed(valid, []byte("qib"))
	valid = appendLenPrefixed(valid, []byte("US"))
	expiry := make([]byte, 8)
	binary.BigEndian.PutUint64(expiry, 200)
	valid = append(valid, expiry...)
	valid = append(valid, 0, 0, 1)
	f.Add(valid)

	invalidBool := append([]byte(nil), valid...)
	invalidBool[len(invalidBool)-1] = 2
	f.Add(invalidBool)

	oversizedField := make([]byte, 12)
	binary.BigEndian.PutUint32(oversizedField[8:12], DRWAMaxFieldBytes+1)
	f.Add(oversizedField)

	f.Fuzz(func(t *testing.T, data []byte) {
		dest := &drwaHolderMirrorView{}
		_ = decodeDRWABinaryHolderMirror(data, dest)
	})
}

func FuzzDecodeDRWABinaryHolderProfile(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 8))
	f.Add(make([]byte, drwaBinaryProfilePayloadMinSize))

	valid := make([]byte, 0, 80)
	valid = append(valid, make([]byte, 8)...)
	valid = appendLenPrefixed(valid, []byte("approved"))
	valid = appendLenPrefixed(valid, []byte("clear"))
	valid = appendLenPrefixed(valid, []byte("accredited"))
	valid = appendLenPrefixed(valid, []byte("GB"))
	expiry := make([]byte, 8)
	binary.BigEndian.PutUint64(expiry, 500)
	valid = append(valid, expiry...)
	f.Add(valid)

	oversizedField := make([]byte, 12)
	binary.BigEndian.PutUint32(oversizedField[8:12], DRWAMaxFieldBytes+1)
	f.Add(oversizedField)

	f.Fuzz(func(t *testing.T, data []byte) {
		dest := &drwaHolderProfileView{}
		_ = decodeDRWABinaryHolderProfile(data, dest)
	})
}

func FuzzDecodeDRWABinaryHolderAuditorAuthorization(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 8))
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0, 1})
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0, 2})

	wrapped := bytes.Repeat([]byte{0xAB}, 9)
	wrapped[8] = 1
	f.Add(wrapped)

	f.Fuzz(func(t *testing.T, data []byte) {
		dest := &drwaHolderAuditorAuthorizationView{}
		_ = decodeDRWABinaryHolderAuditorAuthorization(data, dest)
	})
}
