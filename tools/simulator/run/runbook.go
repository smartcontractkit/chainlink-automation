package run

import (
	"os"

	"github.com/smartcontractkit/chainlink-automation/tools/simulator/config"
)

func LoadSimulationPlan(path string) (config.SimulationPlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config.SimulationPlan{}, err
	}

	return config.DecodeSimulationPlan(data)
}
