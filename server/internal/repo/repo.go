package repo

import "sealdice-mcsm/internal/model"

type Repository interface {
	// SaveBinding saves or updates a binding between an alias and an instance ID.
	SaveBinding(alias, instanceID string) error

	// GetBinding retrieves the instance ID for a given alias.
	// Returns nil, nil if not found.
	GetBinding(alias string) (*model.Binding, error)

	// GetAllBindings returns all bindings.
	GetAllBindings() ([]model.Binding, error)

	// Close closes the repository connection.
	Close() error
}
