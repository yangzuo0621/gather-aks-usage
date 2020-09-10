package pkg

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/build"
)

const (
	patEnvKey = "VSTS_PAT"
	orgUrl    = "https://dev.azure.com/"
)

type buildClient struct {
	organization string
	project      string
}

func (c *buildClient) getPATConnection(ctx context.Context) (*azuredevops.Connection, error) {
	personalAccessToken := os.Getenv(patEnvKey)
	if personalAccessToken == "" {
		return nil, fmt.Errorf("empty VSTS_PAT env")
	}
	connection := azuredevops.NewPatConnection(orgUrl+c.organization, personalAccessToken)
	return connection, nil
}

func (c *buildClient) getBuildClient(ctx context.Context) (build.Client, error) {
	connection, err := c.getPATConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire pat connection: %w", err)
	}

	client, err := build.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("new policy client: %w", err)
	}

	return client, nil
}

func (c *buildClient) GetTopBuildsForPipeline(ctx context.Context, pipelineID int, topN int) ([]build.Build, error) {
	client, err := c.getBuildClient(ctx)
	if err != nil {
		return nil, err
	}

	builds, err := client.GetBuilds(ctx, build.GetBuildsArgs{
		Project:     toStringPtr(c.project),
		Definitions: &[]int{pipelineID},
		Top:         toIntPtr(topN),
	})

	if err != nil {
		return nil, fmt.Errorf("GetTopBuildsForPipeline: %v", err)
	}

	return builds.Value, nil
}

func (c *buildClient) AnalyzeUnderlayTypeFromLogs(ctx context.Context, buildID int) (string, error) {
	client, err := c.getBuildClient(ctx)
	if err != nil {
		return "", err
	}

	timeline, err := client.GetBuildTimeline(ctx, build.GetBuildTimelineArgs{
		Project: toStringPtr(c.project),
		BuildId: toIntPtr(buildID),
	})

	if err != nil {
		return "", fmt.Errorf("AnalyzeUnderlayTypeFromLogs: %v", err)
	}

	logID := -1
	for _, record := range *timeline.Records {
		if *record.Name == "Set underlay type" && record.Log != nil {
			logID = *record.Log.Id
			break
		}
	}

	var underlayType string
	if logID != -1 {
		lines, _ := client.GetBuildLogLines(ctx, build.GetBuildLogLinesArgs{
			Project: toStringPtr(c.project),
			BuildId: toIntPtr(buildID),
			LogId:   toIntPtr(logID),
		})

		for _, s := range *lines {
			index := strings.Index(s, "AKS_E2E_UNDERLAY_TYPE=")
			if index != -1 {
				underlayType = s[index+len("AKS_E2E_UNDERLAY_TYPE")+1:]
				break
			}
		}
	}

	return underlayType, nil
}

func newBuildClient(org string, project string) BuildClient {
	return &buildClient{
		organization: org,
		project:      project,
	}
}

var _ BuildClient = (*buildClient)(nil)
