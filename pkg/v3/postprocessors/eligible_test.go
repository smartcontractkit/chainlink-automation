package postprocessors

import (
	"context"
	"io"
	"log"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/resultstore"
)

func TestNewEligiblePostProcessor(t *testing.T) {
	t.Run("create a new eligible post processor", func(t *testing.T) {
		resultsStore := resultstore.New(log.New(io.Discard, "", 0))
		processor := NewEligiblePostProcessor(resultsStore)

		t.Run("process eligible results", func(t *testing.T) {
			result1 := ocr2keepers.CheckResult{Eligible: false}
			result2 := ocr2keepers.CheckResult{Eligible: true}
			result3 := ocr2keepers.CheckResult{Eligible: false}

			err := processor.PostProcess(context.Background(), []ocr2keepers.CheckResult{
				result1,
				result2,
				result3,
			})

			assert.Nil(t, err)

			storedResults, err := resultsStore.View()
			assert.Nil(t, err)

			assert.Len(t, storedResults, 1)
			assert.True(t, reflect.DeepEqual(storedResults[0], result2))
		})
	})
}
