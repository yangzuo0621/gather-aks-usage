package pkg

import (
	"context"

	"github.com/microsoft/azure-devops-go-api/azuredevops/build"
)

// BuildData encapsulates pipeline and its builds
type BuildData struct {
	Name       string      `json:"name"`
	PipelineID int         `json:"pipeline_id"`
	Builds     []BuildInfo `json:"builds,omitempty"`
}

// BuildInfo encapsulates build information
type BuildInfo struct {
	BuildID      int    `json:"build_id"`
	UnderlayType string `json:"underlay_type"`
	Result       string `json:"result"`
	Time         string `json:"time"`
	URL          string `json:"url"`
	Cluster      string `json:"cluster"`
}

// BuildClient interface for manuplate pipeline
type BuildClient interface {
	// GetTopBuildsForPipeline return top N Builds for specified pipeline
	GetTopBuildsForPipeline(ctx context.Context, pipelineID int, topN int) ([]build.Build, error)

	// AnalyzeUnderlayTypeFromLogs analyzes the underlay type from build logs
	AnalyzeUnderlayTypeFromLogs(ctx context.Context, buildID int) (string, error)

	// AnalyzeClusterFromLogs analyzes the cluster build choosed
	AnalyzeClusterFromLogs(ctx context.Context, buildID int) (string, error)
}
