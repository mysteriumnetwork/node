package openvpn

type optionFlag struct {
	name string
}

func (option *optionFlag) getName() string {
	return option.name
}