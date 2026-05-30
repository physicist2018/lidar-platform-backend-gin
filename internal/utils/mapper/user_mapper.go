package mapper

import (
	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	"github.com/kshmirko/lidar-platform-go/internal/utils/pagination"
	"github.com/kshmirko/lidar-platform-go/pkg/dto"
)

func ToUserResponse(user *entity.User) *dto.UserResponse {
	return &dto.UserResponse{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Role:  string(user.Role),
	}
}

func ToUserResponseList(p *pagination.Pagination[entity.User]) *dto.UserPaginatedResponse {
	items := make([]dto.UserResponse, len(p.Data))
	for i, u := range p.Data {
		items[i] = *ToUserResponse(&u)
	}

	return &dto.UserPaginatedResponse{
		Data:       items,
		Page:       p.Page,
		Limit:      p.Limit,
		TotalItems: p.TotalItems,
		TotalPages: p.TotalPages,
	}
}
