package openvpn

type optionFile struct {
	name string
	path string
}

func (option *optionFile) getName() string {
	return option.name
}