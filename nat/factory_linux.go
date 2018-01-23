package nat

func NewService() NATService {
	return &serviceIPTables{}
}
