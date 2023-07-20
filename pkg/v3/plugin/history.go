package plugin

import ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"

func UpdateHistory(previous *ocr2keepersv3.AutomationOutcome, next *ocr2keepersv3.AutomationOutcome, maxLength int) {
	var (
		historyIdx int
		history    []ocr2keepersv3.BasicOutcome
	)

	if previous != nil {
		historyIdx = previous.NextIdx

		if len(previous.History) < maxLength {
			history = make([]ocr2keepersv3.BasicOutcome, len(previous.History))
			copy(history, previous.History[:len(previous.History)])
		} else {
			history = make([]ocr2keepersv3.BasicOutcome, maxLength)
			copy(history, previous.History[:maxLength])
		}

		// apply the basic outcome to the history
		if len(history) < maxLength {
			// if the history is still less than the limit the new value can be
			// safely appended
			history = append(history, previous.BasicOutcome)
		} else {
			// if the history is at the limit the latest value reset the value at
			// the latest index. this creates a ring buffer of history values
			history[historyIdx] = previous.BasicOutcome
		}

		// advance the latest by 1
		historyIdx++
		if historyIdx >= maxLength {
			historyIdx = 0
		}
	}

	next.History = history
	next.NextIdx = historyIdx
}
