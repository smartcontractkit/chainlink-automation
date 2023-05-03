package logs

import (
	"context"
	"log"
	"time"

	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/smartcontractkit/ocr2keepers/pkg/observer"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

// LogProvider is used by the observer to get upkeep related log events
type LogProvider interface {
	// TODO: TBD range, results type
	GetLogsData(upkeepIDs ...types.UpkeepIdentifier) ([]gethtypes.Log, error)
}

type logTriggerObserver struct {
	logger *log.Logger

	executer    types.Executer
	logProvider LogProvider

	q *LogUpkeepsQueue
}

var _ observer.ObserverV2[time.Time] = &logTriggerObserver{}

func (o *logTriggerObserver) Process(ctx context.Context, t time.Time) {
	upkeeps, checkData, err := o.getExecutableUpkeeps(ctx)
	if err != nil {
		o.logger.Printf("failed to get executable upkeeps: %s", err.Error())
		return
	}
	if len(upkeeps) == 0 {
		o.logger.Println("could not find executable upkeeps")
		return
	}
	results, err := o.executer.Run(ctx, upkeeps, checkData)
	if err != nil {
		o.logger.Printf("failed to execute upkeeps: %s", err.Error())
		return
	}
	o.q.Push(results...)
}

// getExecutableUpkeeps returns a list of upkeeps to execute at the moment and the corresponding check data
func (o *logTriggerObserver) getExecutableUpkeeps(ctx context.Context) ([]types.UpkeepKey, [][]byte, error) {
	var upkeeps []types.UpkeepIdentifier // TODO: populate upkeeps
	logs, err := o.logProvider.GetLogsData(upkeeps...)
	if err != nil {
		return nil, nil, err
	}
	var upkeepKeys []types.UpkeepKey
	var checkData [][]byte
	// TODO: complete
	for _, log := range logs {
		checkData = append(checkData, log.Data)
		// upkeepKeys = append(upkeepKeys, upkeepKey)
	}
	return upkeepKeys, checkData, nil
}

// Propose returns the results that exist in the queue at the moment
func (o *logTriggerObserver) Propose(ctx context.Context) ([]types.UpkeepResult, error) {
	return o.q.Pop(-1), nil
}
