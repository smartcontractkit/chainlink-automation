package observer

import (
	"context"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type Observer interface {
	Observe() (types.BlockKey, []types.UpkeepIdentifier, error)
	CheckUpkeep(ctx context.Context, keys ...types.UpkeepKey) ([]types.UpkeepResult, error)
	Start()
	Stop()
}

// Observer orchestrates the processing of upkeeps and the exposure of triggered upkeeps for the OCR plugin.
// Tick is a generic input the service gets from the corresponding ticker (block/time ticker)
// that invokes the observer.
type ObserverV2[Tick any] interface {
	// Process execute upkeeps, triggered by a generic ticker
	Process(ctx context.Context, t Tick)
	// Propose returns queued upkeep results, called from the OCR plugin upon observation
	Propose(ctx context.Context) ([]types.UpkeepResult, error)
	// Verify executes the given upkeeps
	// decision: not needed for pure reporting
	// Verify(ctx context.Context, keys ...UpkeepKey) (UpkeepResults, error)
}

// RegisterObservers manages a pair of observer/ticker, upon each tick the observer is called
// with the data received from the channel.
func RegisterObservers[T any](
	pctx context.Context,
	cn <-chan T,
	observers ...ObserverV2[T],
) {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	for {
		select {
		case t := <-cn:
			for _, o := range observers {
				// TODO: concurrency control
				go o.Process(ctx, t)
			}
		case <-ctx.Done():
			return
		}
	}
}

// ```mermaid
// sequenceDiagram
//     participant Ticker
//     participant Observer
//     %% TODO: participant Coordinator
//     participant Executer
//     participant Registry
//     participant MercuryLookup
//     participant Encoder

//     par Upkeeps processing
//         Ticker->>Observer: Process(ctx, t)
//         Observer->>Observer: getExecutableUpkeeps()
//         Note over Observer,Observer: NOTE: getExecutableUpkeeps differs between trigger types
//         Observer->>Encoder: EncodeUpkeeps([]upkeep)
//         Encoder-->>Observer: ([]UpkeepKey, []checkData)
//         Observer->>Executer: Execute(ctx, []UpkeepIdentifier, []checkData)
//         loop for each upkeep execute concurrently
//             Executer->>Executer: cache lookup
//             Executer->>Registry: CheckUpkeep([]UpkeepID, [][]checkData)
//             Registry-->>Executer: [[]UpkeepResult]
//             Note over Executer,MercuryLookup: TODO: mercury lookup
//             Executer->>Executer: cache update
//         end
//         Executer-->>Observer: []UpkeepResult
//         Observer->>Observer: Add results to queue
//         Note over Observer,Observer: NOTE: Each adapter maintains a queue of upkeeps results
// end
// ```
