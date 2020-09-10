package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/microsoft/azure-devops-go-api/azuredevops/build"
	"github.com/spf13/cobra"
)

const (
	organization    = "msazure"
	project         = "CloudNativeCompute"
	aksUnderlayType = "AKS_CLUSTER"
)

// CreateCommand creates an instance command cli
func CreateCommand() *cobra.Command {
	c := &cobra.Command{
		Use:          "gather-aks-usage",
		Short:        "Gather the detailed usage of AKS as underlay",
		SilenceUsage: true,
	}

	c.AddCommand(createCountCommand())
	return c
}

func createCountCommand() *cobra.Command {
	var (
		jsonFile string
		topN     int
		datas    []Data
	)

	c := &cobra.Command{
		Use:          "count",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
				return err
			}
			file, _ := ioutil.ReadFile(jsonFile)
			json.Unmarshal(file, &datas)

			client := newBuildClient(organization, project)
			if topN == 0 {
				topN = 10
			}

			for index := range datas {
				builds, err := client.GetTopBuildsForPipeline(ctx, datas[index].PipelineID, topN)
				if err != nil {
					log.Fatalf("%v", err)
					return err
				}

				for _, b := range builds {
					if *b.Status == build.BuildStatusValues.Completed {
						underlayType, _ := client.AnalyzeUnderlayTypeFromLogs(ctx, *b.Id)

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
			err := encoder.Encode(datas)
			if err != nil {
				return err
			}

			return ioutil.WriteFile(jsonFile, b.Bytes(), 0644)
		},
	}

	c.Flags().StringVar(&jsonFile, "file", "", "model file")
	c.Flags().IntVar(&topN, "top", 0, "top N records")
	c.MarkFlagRequired("file")

	return c
}