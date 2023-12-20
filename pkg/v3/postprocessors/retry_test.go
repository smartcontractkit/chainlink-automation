package postprocessors

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/stores"
	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

func TestRetryPostProcessor_PostProcess(t *testing.T) {
	lggr := log.Default()
	q := stores.NewRetryQueue(lggr)
	processor := NewRetryablePostProcessor(q, lggr)

	results := []ocr2keepers.CheckResult{
		{Retryable: true, PipelineExecutionState: 1},
		{Retryable: false, PipelineExecutionState: 3},
		{Retryable: true, RetryInterval: time.Second, PipelineExecutionState: 2},
	}

	// Call the PostProcess method
	err := processor.PostProcess(context.Background(), results, []ocr2keepers.UpkeepPayload{
		{WorkID: "1"}, {WorkID: "2"}, {WorkID: "3"},
	})
	assert.Nil(t, err, "PostProcess returned an error: %v", err)

	assert.Equal(t, 2, q.Size())
}
