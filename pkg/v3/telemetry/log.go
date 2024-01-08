package telemetry

import (
	"fmt"
	"io"
	"log"
	"time"
)

type Status int

const (
	Surfaced Status = iota
	CheckPipelineRun
	Queued
	Proposed
	AgreedInQuorum
	ResultProposed
	ResultAgreedInQuorum
	Reported
	Completed
)

const (
	ServiceName    = "automation-ocr3"
	LogPkgStdFlags = log.Lshortfile
)

func WrapLogger(logger *log.Logger, ns string) *log.Logger {
	return log.New(logger.Writer(), fmt.Sprintf("[%s | %s]", ServiceName, ns), LogPkgStdFlags)
}

func WrapTelemetryLogger(logger *Logger, ns string) *Logger {
	baseLogger := log.New(logger.Writer(), fmt.Sprintf("[%s | %s]", ServiceName, ns), LogPkgStdFlags)

	return &Logger{
		Logger:    baseLogger,
		collector: logger.collector,
	}
}

type Logger struct {
	*log.Logger
	collector io.Writer
}

func NewTelemetryLogger(logger *log.Logger, collector io.Writer) *Logger {
	return &Logger{
		Logger:    logger,
		collector: collector,
	}
}

func (l *Logger) Collect(key string, block uint64, status Status) error {
	_, err := l.collector.Write([]byte(fmt.Sprintf(`{"key":"%s","block":%d,"status":%d,"time":"%s"}`, key, block, status, time.Now().Format(time.RFC3339Nano))))

	return err
}

func (l *Logger) GetLogger() *log.Logger {
	return l.Logger
}
