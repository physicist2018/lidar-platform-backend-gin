package processing

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

// Processor is the interface that every algorithm processor must implement.
type Processor interface {
	// Name returns the algorithm name (e.g. "stage0", "stage1").
	Name() string
	// Execute runs the processing algorithm.
	// The run.Params will be parsed by each implementation according to its own parameter struct.
	Execute(ctx context.Context, run *entity.ProcessingRun) error
}
