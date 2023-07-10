package run

import (
	"os"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
)

func LoadRunBook(path string) (config.RunBook, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config.RunBook{}, err
	}

	return config.LoadRunBook(data)
}
