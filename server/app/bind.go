package app

// Bind binds an instance ID to an alias.
func (app *Application) Bind(alias, instanceID string) error {
	return app.Repo.SaveBinding(alias, instanceID)
}
