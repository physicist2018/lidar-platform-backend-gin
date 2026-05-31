package mapper

import (
	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	"github.com/kshmirko/lidar-platform-go/internal/utils/pagination"
	"github.com/kshmirko/lidar-platform-go/pkg/dto"
)

func ToExperimentResponse(exp *entity.Experiment) *dto.ExperimentResponse {
	return &dto.ExperimentResponse{
		ID:                   exp.ID,
		Title:                exp.Title,
		Comments:             exp.Comments,
		MeasurementStartTime: exp.MeasurementStartTime,
		MeasurementStopTime:  exp.MeasurementStopTime,
		LicelZipPath:         exp.LicelZipPath,
		LicelBgrPath:         exp.LicelBgrPath,
		MeteoFilePath:        exp.MeteoFilePath,
		Status:               string(exp.Status),
		ErrorMsg:             exp.ErrorMsg,
		CreatedAt:            exp.CreatedAt,
		UpdatedAt:            exp.UpdatedAt,
	}
}

func ToExperimentResponseList(p *pagination.Pagination[entity.Experiment]) *dto.ExperimentPaginatedResponse {
	items := make([]dto.ExperimentResponse, len(p.Data))
	for i, exp := range p.Data {
		items[i] = *ToExperimentResponse(&exp)
	}
	return &dto.ExperimentPaginatedResponse{
		Data:       items,
		Page:       p.Page,
		Limit:      p.Limit,
		TotalItems: p.TotalItems,
		TotalPages: p.TotalPages,
	}
}
