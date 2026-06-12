package mapper

import (
	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/pkg/dto"
)

func ToProcessingRunResponse(run *entity.ProcessingRun) *dto.ProcessingRunResponse {
	return &dto.ProcessingRunResponse{
		ID:           run.ID,
		ExperimentID: run.ExperimentID,
		Algorithm:    run.Algorithm,
		Params:       run.Params,
		Status:       string(run.Status),
		ErrorMsg:     run.ErrorMsg,
		CreatedAt:    run.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    run.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
