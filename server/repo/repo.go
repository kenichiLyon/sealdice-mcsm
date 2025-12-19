package repo

type BindingRepo interface {
	SaveBinding(alias, instanceID string) error
	GetBinding(alias string) (string, error)
	GetAllBindings() (map[string]string, error)
	Close() error
}
