package commands

type UninstallHook struct{}

func (command UninstallHook) Execute([]string) error {
	return nil
}
