package pagination

import "math"

type Pagination[T any] struct {
	Data       []T   `json:"data"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

func New[T any](data []T, totalItems int64, page, limit int) *Pagination[T] {
	if data == nil {
		data = make([]T, 0)
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))

	return &Pagination[T]{
		Data:       data,
		Page:       page,
		Limit:      limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}

func Offset(page, limit int) int {
	return (page - 1) * limit
}
