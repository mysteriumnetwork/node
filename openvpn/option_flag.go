package openvpn

func OptionFlag(name string) optionFlag {
	return optionFlag{name}
}

type optionFlag struct {
	name string
}

func (option optionFlag) getName() string {
	return option.name
}

func (option optionFlag) toArguments(arguments *[]string) error {
	*arguments = append(*arguments, "--" + option.name)
	return nil
}

func (option optionFlag) toFile() (string, error) {
	return option.name, nil
}