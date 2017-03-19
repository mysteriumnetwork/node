package openvpn

type optionParam struct {
	name string
	value string
}

func (option *optionParam) getName() string {
	return option.name
}