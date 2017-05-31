package command_run

type Command interface {
	Run(options CommandOptions) error
	Wait() error
	Kill()
}
