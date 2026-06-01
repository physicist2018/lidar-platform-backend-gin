package mapper

import (
	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	"github.com/kshmirko/lidar-platform-go/pkg/dto"
)

func ToPreparedExperimentResponse(exp *entity.PreparedExperiment) *dto.PreparedExperimentResponse {
	return &dto.PreparedExperimentResponse{
		ID:           exp.ID,
		UserID:       exp.UserID,
		ExperimentID: exp.ExperimentID,
		CropAlt:      exp.CropAlt,
		BGRType:      string(exp.BGRType),
		BGRAlt:       exp.BGRAlt,
		PathToData:   exp.PathToData,
		Status:       string(exp.Status),
		ErrorMsg:     exp.ErrorMsg,
	}
}
