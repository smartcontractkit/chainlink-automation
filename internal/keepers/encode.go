package keepers

import (
	"bytes"
	"encoding/gob"
)

// Encode is a convenience method that uses gob encoding to
// encode any value to an array of bytes
func Encode[T any](value T) ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)

	err := enc.Encode(value)
	if err != nil {
		return []byte{}, err
	}

	return b.Bytes(), nil
}

// Decode is a convenience method that uses gob encoding to
// decode any value from an array of bytes
func Decode[T any](b []byte, value *T) error {
	bts := bytes.NewReader(b)
	dec := gob.NewDecoder(bts)
	return dec.Decode(value)
}
