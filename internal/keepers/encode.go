package keepers

import (
	"bytes"
	"encoding/json"
)

// Encode is a convenience method that uses json encoding to
// encode any value to an array of bytes
// gob encoding was compared to json encoding where gob encoding
// was shown to be 8x slower
func Encode[T any](value T) ([]byte, error) {
	var b bytes.Buffer

	enc := json.NewEncoder(&b)
	err := enc.Encode(value)
	if err != nil {
		return []byte{}, err
	}

	return b.Bytes(), nil
}

// Decode is a convenience method that uses json encoding to
// decode any value from an array of bytes
func Decode[T any](b []byte, value *T) error {
	bts := bytes.NewReader(b)
	dec := json.NewDecoder(bts)
	return dec.Decode(value)
}
