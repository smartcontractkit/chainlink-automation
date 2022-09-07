package main

type OCRReceiver struct {
	Name         string
	Init         chan OCRCall[struct{}]
	Query        chan OCRCall[OCRQuery]
	Observations chan OCRCall[[]OCRObservation]
	Report       chan OCRCall[OCRReport]
	Stop         chan struct{}
}

func NewOCRReceiver(name string) *OCRReceiver {
	return &OCRReceiver{
		Name:         name,
		Init:         make(chan OCRCall[struct{}], 1),
		Query:        make(chan OCRCall[OCRQuery], 1),
		Observations: make(chan OCRCall[[]OCRObservation], 1),
		Report:       make(chan OCRCall[OCRReport], 1),
		Stop:         make(chan struct{}),
	}
}
