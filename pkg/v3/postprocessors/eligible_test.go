package postprocessors

import (
	"context"
	"io"
	"log"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/stores"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

func TestNewEligiblePostProcessor(t *testing.T) {
	resultsStore := stores.New(log.New(io.Discard, "", 0))
	processor := NewEligiblePostProcessor(resultsStore, log.New(io.Discard, "", 0))

	t.Run("process eligible results", func(t *testing.T) {
		result1 := ocr2keepers.CheckResult{Eligible: false}
		result2 := ocr2keepers.CheckResult{Eligible: true}
		result3 := ocr2keepers.CheckResult{Eligible: false}

		err := processor.PostProcess(context.Background(), []ocr2keepers.CheckResult{
			result1,
			result2,
			result3,
		}, []ocr2keepers.UpkeepPayload{
			{WorkID: "1"},
			{WorkID: "2"},
			{WorkID: "3"},
		})

		assert.Nil(t, err)

		storedResults, err := resultsStore.View()
		assert.Nil(t, err)

		assert.Len(t, storedResults, 1)
		assert.True(t, reflect.DeepEqual(storedResults[0], result2))
	})
}
