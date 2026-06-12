package mapper

import (
	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/pkg/dto"
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
