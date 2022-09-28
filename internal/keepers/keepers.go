package keepers

import (
	"log"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type keepers struct {
	service upkeepService
	encoder types.ReportEncoder
	logger  *log.Logger
}

/*
type offChainConfig struct {
}

type onChainConfig struct {
}
*/
