package dto

import (
	"github.com/physicist2018/lidar-platform-go/internal/utils/pagination"
)

type ExperimentPaginatedResponse = pagination.Pagination[ExperimentResponse]
