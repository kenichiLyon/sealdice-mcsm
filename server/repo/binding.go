package repo

// Binding represents the mapping between an alias and an MCSM instance ID.
type Binding struct {
	Alias      string `json:"alias"`
	InstanceID string `json:"instance_id"`
}

type Repository interface {
	// SaveBinding saves or updates a binding between an alias and an instance ID.
	SaveBinding(alias, instanceID string) error

	// GetBinding retrieves the instance ID for a given alias.
	// Returns nil, nil if not found.
	GetBinding(alias string) (*Binding, error)

	// GetAllBindings returns all bindings.
	GetAllBindings() ([]Binding, error)

	// Close closes the repository connection.
	Close() error
}
