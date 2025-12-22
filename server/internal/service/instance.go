package service

import (
	"sealdice-mcsm/server/internal/data"
)

type InstanceService struct {
	repo data.BindingRepo
}

func NewInstanceService(repo data.BindingRepo) *InstanceService {
	return &InstanceService{repo: repo}
}

func (s *InstanceService) Bind(alias, protocolID, coreID string) error {
	return s.repo.SaveBinding(&data.Binding{
		Alias:              alias,
		ProtocolInstanceID: protocolID,
		CoreInstanceID:     coreID,
	})
}

func (s *InstanceService) Unbind(alias string) error {
	return s.repo.DeleteBinding(alias)
}

func (s *InstanceService) GetByAlias(alias string) (*data.Binding, error) {
	return s.repo.GetBinding(alias)
}

func (s *InstanceService) GetAll() ([]*data.Binding, error) {
	return s.repo.GetAllBindings()
}
