package observer

type simpleService struct {
	f func() error
	c func()
}

func (sw *simpleService) Do() error {
	return sw.f()
}

func (sw *simpleService) Stop() {
	sw.c()
}
