package nat

func NewService() NATService {
	return &serviceFake{}
}
