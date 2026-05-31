package dto

import (
	"github.com/kshmirko/lidar-platform-go/internal/utils/pagination"
)

type ExperimentPaginatedResponse = pagination.Pagination[ExperimentResponse]
