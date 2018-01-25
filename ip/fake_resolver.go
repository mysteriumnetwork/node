package ip

func NewFakeResolver(ip string) Resolver {
	return &fakeResolver{
		ip:    ip,
		error: nil,
	}
}

func NewFailingFakeResolver(err error) Resolver {
	return &fakeResolver{
		ip:    "",
		error: err,
	}
}

type fakeResolver struct {
	ip    string
	error error
}

func (client *fakeResolver) GetPublicIP() (string, error) {
	return client.ip, client.error
}

func (client *fakeResolver) GetOutboundIP() (string, error) {
	return client.ip, client.error
}
