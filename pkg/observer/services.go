package observer

type SimpleService struct {
	F func() error
	C func()
}

func NewSimpleService(f func() error, c func()) *SimpleService {
	if f == nil {
		f = func() error { return nil }
	}
	if c == nil {
		c = func() {}
	}
	return &SimpleService{
		F: f,
		C: c,
	}
}

func (sw *SimpleService) Do() error {
	return sw.F()
}

func (sw *SimpleService) Stop() {
	sw.C()
}
