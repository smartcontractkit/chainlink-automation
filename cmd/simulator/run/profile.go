package run

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

type ProfilerConfig struct {
	// Enabled true starts the profiler
	Enabled bool
	// PprofPort is the port to listen for pprof input
	PprofPort int
	// Wait is the time to wait for the profiler to start before moving on
	Wait time.Duration
}

func Profiler(config ProfilerConfig, logger *log.Logger) {
	if config.Enabled {
		if logger != nil {
			logger.Println("starting profiler; waiting 5 seconds to start simulation")
		}

		go func() {
			err := http.ListenAndServe(fmt.Sprintf("localhost:%d", config.PprofPort), nil)
			if logger != nil && err != nil {
				logger.Printf("pprof listener returned error on exit: %s", err)
			}
		}()

		if config.Wait > 0 {
			time.Sleep(config.Wait)
		}
	}
}
