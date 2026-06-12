package mapper

import (
	domain "github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	dbEntity "github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/entity"
)

func ToUserDomain(e *dbEntity.UserEntity) *domain.User {
	return &domain.User{
		ID:       e.ID,
		Name:     e.Name,
		Email:    e.Email,
		Role:     domain.UserRole(e.Role),
		Password: e.Password,
	}
}

func ToUserDomainList(entities []dbEntity.UserEntity) []domain.User {
	result := make([]domain.User, len(entities))
	for i, e := range entities {
		result[i] = domain.User{
			ID:       e.ID,
			Name:     e.Name,
			Email:    e.Email,
			Role:     domain.UserRole(e.Role),
			Password: e.Password,
		}
	}
	return result
}
