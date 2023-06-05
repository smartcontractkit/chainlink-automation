package postprocessors

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type checkResult struct {
	eligible bool
	data     string
}

func (r *checkResult) IsEligible() bool {
	return r.eligible
}

func TestNewEligiblePostProcessor(t *testing.T) {
	t.Run("create a new eligible post processor", func(t *testing.T) {
		resultsStore := ocr2keepers.NewResultStore[ocr2keepers.CheckResult]()
		processor := NewEligiblePostProcessor(resultsStore)

		t.Run("process eligible results", func(t *testing.T) {
			result1 := &checkResult{eligible: false, data: "result 1 data"}
			result2 := &checkResult{eligible: true, data: "result 2 data"}
			result3 := &checkResult{eligible: false, data: "result 3 data"}

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
