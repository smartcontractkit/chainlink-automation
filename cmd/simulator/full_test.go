package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeValidConfig() *SimulatorConfig {
	addr := "0x02777053d6764996e594c3E88AF1D58D5363a2e6"
	rpc := "https://some.value.test/place"
	nodes := 1
	return &SimulatorConfig{
		ContractAddress: &addr,
		RPC:             &rpc,
		Nodes:           &nodes,
	}
}

func TestValidateSimulatorConfig(t *testing.T) {
	conf := makeValidConfig()
	err := validateSimulatorConfig(conf)
	assert.Nil(t, err)

	t.Run("Valid Contract Required", func(t *testing.T) {
		c := makeValidConfig()
		c.ContractAddress = nil
		err = validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)

		value := "test"
		c.ContractAddress = &value
		err = validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)
	})

	t.Run("RPC Required", func(t *testing.T) {
		c := makeValidConfig()
		c.RPC = nil
		err = validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)
	})

	t.Run("Report Output Path Exists if Defined", func(t *testing.T) {
		c := makeValidConfig()
		path := "./path-should-not-exists-locally"
		c.ReportOutputPath = &path
		err := validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)

		path = "../../internal"
		err = validateSimulatorConfig(c)
		assert.Nil(t, err)
	})

	t.Run("Node Count Required and Greater Than 0", func(t *testing.T) {
		c := makeValidConfig()
		c.Nodes = nil
		err := validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)

		nodes := -1
		c.Nodes = &nodes
		err = validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)

		nodes = 0
		err = validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)
	})

	t.Run("Rounds Must be Greater Than or Equal to 0 if Defined", func(t *testing.T) {
		c := makeValidConfig()
		rds := -1
		c.Rounds = &rds
		err := validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)

		rds = 0
		err = validateSimulatorConfig(c)
		assert.Nil(t, err)
	})

	t.Run("Round Time Must be Greater Than or Equal to 0 if Defined", func(t *testing.T) {
		c := makeValidConfig()
		rds := -1
		c.RoundTime = &rds
		err := validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)

		rds = 0
		err = validateSimulatorConfig(c)
		assert.Nil(t, err)
	})

	t.Run("Query Time Must be Greater Than or Equal to 0 if Defined", func(t *testing.T) {
		c := makeValidConfig()
		rds := -1
		c.QueryTimeLimit = &rds
		err := validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)

		rds = 0
		err = validateSimulatorConfig(c)
		assert.Nil(t, err)
	})

	t.Run("Observation Time Must be Greater Than or Equal to 0 if Defined", func(t *testing.T) {
		c := makeValidConfig()
		rds := -1
		c.ObservationTimeLimit = &rds
		err := validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)

		rds = 0
		err = validateSimulatorConfig(c)
		assert.Nil(t, err)
	})

	t.Run("Report Time Must be Greater Than or Equal to 0 if Defined", func(t *testing.T) {
		c := makeValidConfig()
		rds := -1
		c.ObservationTimeLimit = &rds
		err := validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)

		rds = 0
		err = validateSimulatorConfig(c)
		assert.Nil(t, err)
	})

	t.Run("Max Run Time Must be Greater Than or Equal to 0 if Defined", func(t *testing.T) {
		c := makeValidConfig()
		rds := -1
		c.MaxRunTime = &rds
		err := validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)

		rds = 0
		err = validateSimulatorConfig(c)
		assert.Nil(t, err)
	})

	t.Run("Round Limit or Function Limits", func(t *testing.T) {
		var err error
		c := makeValidConfig()
		tm := 1

		// valid
		c.RoundTime = &tm
		err = validateSimulatorConfig(c)
		assert.Nil(t, err)

		// unset the first and set another: valid
		c.RoundTime = nil
		c.QueryTimeLimit = &tm
		err = validateSimulatorConfig(c)
		assert.Nil(t, err)

		// any of the function limits: valid
		c.QueryTimeLimit = nil
		c.ObservationTimeLimit = &tm
		err = validateSimulatorConfig(c)
		assert.Nil(t, err)

		// last check on functions: valid
		c.ObservationTimeLimit = nil
		c.ReportTimeLimit = &tm
		err = validateSimulatorConfig(c)
		assert.Nil(t, err)

		// round time cannot be set with function limits: invalid
		c.RoundTime = &tm
		err = validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)

		// invalid
		c.ReportTimeLimit = nil
		c.ObservationTimeLimit = &tm
		err = validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)

		// invalid
		c.ObservationTimeLimit = nil
		c.QueryTimeLimit = &tm
		err = validateSimulatorConfig(c)
		assert.ErrorIs(t, err, ErrInvalidConfig)
	})
}
