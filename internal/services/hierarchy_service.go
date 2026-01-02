package services

import (
	"github.com/shigake/tech-iq-back/internal/repositories"

	"github.com/google/uuid"
)

type HierarchyService struct {
	hierarchyRepo repositories.HierarchyRepository
}

func NewHierarchyService(hierarchyRepo repositories.HierarchyRepository) *HierarchyService {
	return &HierarchyService{hierarchyRepo: hierarchyRepo}
}

// GetUserScopes retorna os IDs dos escopos que um usuário tem acesso
func (s *HierarchyService) GetUserScopes(userID uuid.UUID) ([]uuid.UUID, error) {
	// Buscar todos os nodes que o usuário é membro
	memberships, err := s.hierarchyRepo.GetUserMemberships(userID)
	if err != nil {
		return nil, err
	}

	scopeIDs := make([]uuid.UUID, 0)
	for _, m := range memberships {
		if m.NodeID != uuid.Nil {
			scopeIDs = append(scopeIDs, m.NodeID)
		}
	}

	return scopeIDs, nil
}

// CanViewNode verifica se um usuário pode visualizar um node específico
func (s *HierarchyService) CanViewNode(userID, nodeID uuid.UUID) (bool, error) {
	scopes, err := s.GetUserScopes(userID)
	if err != nil {
		return false, err
	}

	for _, scopeID := range scopes {
		if scopeID == nodeID {
			return true, nil
		}
		// TODO: Verificar se nodeID é descendente de scopeID
	}

	return false, nil
}
