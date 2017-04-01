package openvpn

func OptionParam(name, value string) optionParam {
	return optionParam{name, value}
}

type optionParam struct {
	name  string
	value string
}

func (option optionParam) getName() string {
	return option.name
}

func (option optionParam) toCli() (string, error) {
	return "--" + option.name + " " + option.value, nil
}

func (option optionParam) toFile() (string, error) {
	return option.name + " " + option.value, nil
}
