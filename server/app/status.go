package app

// Status returns the status of an instance or dashboard.
func (app *Application) Status(target string) (interface{}, error) {
	if target == "" {
		// Return MCSM Dashboard
		return app.MCSM.GetDashboard()
	}

	id, err := app.resolveTarget(target)
	if err != nil {
		return nil, err
	}
	statusData, err := app.MCSM.GetInstanceStatus(id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"target": target,
		"id":     id,
		"data":   statusData,
	}, nil
}
