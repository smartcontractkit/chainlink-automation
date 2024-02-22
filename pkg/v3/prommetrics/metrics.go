package prommetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// AutomationNamespace is the namespace for all Automation related metrics
const AutomationNamespace = "automation"

// Automation metrics
var (
	AutomationNewResultsAddedFromPerformables = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: AutomationNamespace,
		Name:      "new_results_added_from_performables",
		Help:      "How many results were added from the peformables for a given observation",
	})
	AutomationTotalPerformablesInObservation = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: AutomationNamespace,
		Name:      "total_performables_in_observation",
		Help:      "How many total performables were in the observation",
	})
	AutomationErrorInvalidOracleObservation = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: AutomationNamespace,
		Name:      "error_invalid_oracle_observation",
		Help:      "Count of how many invalid oracle observations have been made",
	})
	AutomationErrorPreviousOutcomeDecode = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: AutomationNamespace,
		Name:      "error_previous_outcome_decode",
		Help:      "Count of how many errors were encountered when decoding previous outcome",
	})
)
