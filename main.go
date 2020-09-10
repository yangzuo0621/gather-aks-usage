package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/build"
)

const (
	patEnvKey       = "VSTS_PAT"
	orgUrl          = "https://dev.azure.com/"
	organization    = "msazure"
	project         = "CloudNativeCompute"
	top             = 30
	aksUnderlayType = "AKS_CLUSTER"
)

var datas []Data

func main() {
	conn, err := getPATConnection(organization)
	handleError(err)

	ctx := context.Background()

	client, err := getBuildClient(ctx, conn)
	handleError(err)

	file, _ := ioutil.ReadFile("result.json")
	json.Unmarshal(file, &datas)

	for index := range datas {
		builds, err := getTopBuildsForPipeline(ctx, client, project, datas[index].PipelineID)
		handleError(err)

		for _, b := range builds {
			if *b.Status == build.BuildStatusValues.Completed {
				underlayType, err := analyzeUnderlayTypeFromLogs(ctx, client, project, *b.Id)
				handleError(err)

				if underlayType == aksUnderlayType {
					if datas[index].Builds == nil {
						datas[index].Builds = make(map[int]BuildInfo)
					}

					if _, ok := datas[index].Builds[*b.Id]; !ok {
						datas[index].Builds[*b.Id] = BuildInfo{
							BuildID:      *b.Id,
							UnderlayType: underlayType,
							Result:       string(*b.Result),
							Time:         b.FinishTime.Time.Local().Format("2006-01-02"),
							URL:          fmt.Sprintf("https://dev.azure.com/%s/%s/_build/results?buildId=%d&view=results", organization, project, *b.Id),
						}
						fmt.Println("not exist, add it")
					}
				}

				fmt.Println("BuildNumber=", *b.Id, ", status=", *b.Status, ", result=", *b.Result, ", type=", underlayType, ", pipelineID=", datas[index].PipelineID)
			} else {
				fmt.Println("BuildNumber=", *b.Id, ", status=", *b.Status, "skip it")
			}
		}
	}

	b := new(bytes.Buffer)
	encoder := json.NewEncoder(b)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", " ")
	encoder.Encode(datas)

	_ = ioutil.WriteFile("result.json", b.Bytes(), 0644)
}

func getPATConnection(organization string) (*azuredevops.Connection, error) {
	pat := os.Getenv(patEnvKey)
	if pat == "" {
		return nil, fmt.Errorf("empty VSTS_PAT env")
	}

	connection := azuredevops.NewPatConnection(orgUrl+organization, pat)
	return connection, nil
}

func getBuildClient(ctx context.Context, connection *azuredevops.Connection) (build.Client, error) {
	client, err := build.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("getBuildClient: %v", err)
	}
	return client, nil
}

func getTopBuildsForPipeline(ctx context.Context, client build.Client, project string, pipelineID int) ([]build.Build, error) {
	builds, err := client.GetBuilds(ctx, build.GetBuildsArgs{
		Project:     toStringPtr(project),
		Definitions: &[]int{pipelineID},
		Top:         toIntPtr(top),
	})

	if err != nil {
		return nil, fmt.Errorf("getTopTenBuildsForPipeline: %v", err)
	}

	return builds.Value, nil
}

func analyzeUnderlayTypeFromLogs(ctx context.Context, client build.Client, project string, buildID int) (string, error) {
	// search pattern: AKS_E2E_UNDERLAY_TYPE=AKS_ENGINE_CLUSTER
	timeline, err := client.GetBuildTimeline(ctx, build.GetBuildTimelineArgs{
		Project: toStringPtr(project),
		BuildId: toIntPtr(buildID),
	})

	if err != nil {
		return "", fmt.Errorf("analyzeUnderlayTypeFromLogs: %v", err)
	}

	logID := -1
	for _, record := range *timeline.Records {
		if *record.Name == "Set underlay type" {
			logID = *record.Log.Id
			break
		}
	}

	var underlayType string
	if logID != -1 {
		lines, _ := client.GetBuildLogLines(ctx, build.GetBuildLogLinesArgs{
			Project: toStringPtr(project),
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

func toStringPtr(s string) *string {
	return &s
}

func toIntPtr(i int) *int {
	return &i
}

func handleError(err error) {
	if err != nil {
		log.Fatalf("%v", err)
		os.Exit(-1)
	}
}
