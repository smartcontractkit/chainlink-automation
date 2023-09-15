package config

import (
	"encoding/json"
	"fmt"
	"time"
)

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	p, err := time.ParseDuration(raw)
	if err != nil {
		return err
	}

	*d = Duration(p)
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	str := fmt.Sprintf(`"%s"`, time.Duration(d).String())

	return []byte(str), nil
}

func (d Duration) Value() time.Duration {
	return time.Duration(d)
}
