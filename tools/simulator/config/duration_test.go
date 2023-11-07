package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDurationEncoding(t *testing.T) {
	rawValue := `"300ms"`

	var tm Duration
	err := json.Unmarshal([]byte(rawValue), &tm)

	require.NoError(t, err, "no error expected from unmarshalling")

	value, err := tm.MarshalJSON()

	require.NoError(t, err, "no error expected from marshalling")

	assert.Equal(t, rawValue, string(value))
}
