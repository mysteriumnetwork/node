package nat

func NewService() NATService {
	return &serviceIpTables{}
}
