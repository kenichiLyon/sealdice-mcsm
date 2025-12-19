package app

import "fmt"

// Control performs an action on an instance.
func (app *Application) Control(target, action string) error {
	id, err := app.resolveTarget(target)
	if err != nil {
		return err
	}

	switch action {
	case "start":
		return app.MCSM.StartInstance(id)
	case "stop":
		return app.MCSM.StopInstance(id)
	case "restart":
		return app.MCSM.RestartInstance(id)
	case "fstop":
		return app.MCSM.ForceStopInstance(id)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}
