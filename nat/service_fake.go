package nat

func NewServiceFake() *serviceFake {
	return &serviceFake{}
}

type serviceFake struct {
}

func (service *serviceFake) Add(rule RuleForwarding) {

}

func (service *serviceFake) Start() error {
	return nil
}

func (service *serviceFake) Stop() error {
	return nil
}
