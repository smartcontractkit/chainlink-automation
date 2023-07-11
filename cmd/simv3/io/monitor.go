package io

import "io"

type MonitorToWriter struct {
	w io.Writer
}

func NewMonitorToWriter(w io.Writer) *MonitorToWriter {
	return &MonitorToWriter{
		w: w,
	}
}

func (m *MonitorToWriter) SendLog(log []byte) {
	_, _ = m.w.Write(log)
}
