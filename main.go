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
	"github.com/spf13/cobra"
)

const (
	patEnvKey       = "VSTS_PAT"
	orgUrl          = "https://dev.azure.com/"
	organization    = "msazure"
	project         = "CloudNativeCompute"
	top             = 30
	aksUnderlayType = "AKS_CLUSTER"
)

var (
	datas    []Data
	jsonFile string
)

func main() {
	log.SetOutput(os.Stdout)
	command := &cobra.Command{
		Use: "gather-aks-usage",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
				log.Fatalf("%v", err)
				return err
			}

			conn, err := getPATConnection(organization)
			if err != nil {
				log.Fatalf("%v", err)
				return err
			}

			ctx := context.Background()

			client, err := getBuildClient(ctx, conn)
			if err != nil {
				log.Fatalf("%v", err)
				return err
			}

			file, _ := ioutil.ReadFile(jsonFile)
			json.Unmarshal(file, &datas)

			for index := range datas {
				builds, err := getTopBuildsForPipeline(ctx, client, project, datas[index].PipelineID)
				if err != nil {
					log.Fatalf("%v", err)
					return err
				}

				for _, b := range builds {
					if *b.Status == build.BuildStatusValues.Completed {
						underlayType, err := analyzeUnderlayTypeFromLogs(ctx, client, project, *b.Id)
						if err != nil {
							log.Fatalf("%v", err)
							return err
						}

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
								log.Println("not exist, add it")
							}
						}

						log.Println("BuildNumber=", *b.Id, ", status=", *b.Status, ", result=", *b.Result, ", type=", underlayType, ", pipelineID=", datas[index].PipelineID)
					} else {
						log.Println("BuildNumber=", *b.Id, ", status=", *b.Status, "skip it")
					}
				}
			}

			b := new(bytes.Buffer)
			encoder := json.NewEncoder(b)
			encoder.SetEscapeHTML(false)
			encoder.SetIndent("", " ")
			err = encoder.Encode(datas)
			if err != nil {
				return err
			}

			log.Println("Finish")

			return ioutil.WriteFile(jsonFile, b.Bytes(), 0644)
		},
	}

	command.Flags().StringVar(&jsonFile, "file", "", "model file")
	command.MarkFlagRequired("file")

	if err := command.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
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
