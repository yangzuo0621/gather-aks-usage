package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"

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
	c.AddCommand(createOutputCommand())
	return c
}

func createCountCommand() *cobra.Command {
	var (
		jsonFile string
		topN     int
		datas    []BuildData
	)

	c := &cobra.Command{
		Use:          "count",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.SetOutput(os.Stdout)
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
						cluster, _ := client.AnalyzeClusterFromLogs(ctx, *b.Id)

						if underlayType == aksUnderlayType {
							if datas[index].Builds == nil {
								datas[index].Builds = []BuildInfo{}
							}

							exists := false
							for _, d := range datas[index].Builds {
								if d.BuildID == *b.Id {
									exists = true
									break
								}
							}

							if exists != true {
								datas[index].Builds = append(datas[index].Builds, BuildInfo{
									BuildID:      *b.Id,
									UnderlayType: underlayType,
									Result:       string(*b.Result),
									Time:         b.FinishTime.Time.Local().Format("2006-01-02"),
									URL:          fmt.Sprintf("https://dev.azure.com/%s/%s/_build/results?buildId=%d&view=results", organization, project, *b.Id),
									Cluster:      cluster,
								})
								log.Println("not exist, add it")
							}
						}

						log.Println("BuildNumber=", *b.Id, ", status=", *b.Status, ", result=", *b.Result, ", type=", underlayType, ", pipelineID=", datas[index].PipelineID, "cluster=", cluster)
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

			log.Println("Finish")

			return ioutil.WriteFile(jsonFile, b.Bytes(), 0644)
		},
	}

	c.Flags().StringVar(&jsonFile, "file", "", "model file")
	c.Flags().IntVar(&topN, "top", 0, "top N records")
	c.MarkFlagRequired("file")

	return c
}

func createOutputCommand() *cobra.Command {
	var (
		jsonFile string
		datas    []BuildData
	)

	c := &cobra.Command{
		Use:          "output",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
				return err
			}
			file, _ := ioutil.ReadFile(jsonFile)
			json.Unmarshal(file, &datas)

			for _, data := range datas {
				fmt.Println("============================")
				fmt.Printf("| %-8d | %s |  \n", data.PipelineID, data.Name)

				builds := data.Builds
				sort.Slice(builds[:], func(i, j int) bool {
					return builds[i].Time < builds[j].Time
				})

				for _, v := range builds {
					fmt.Printf("| %d | %s | %s | %s | %-9s | %s |\n", v.BuildID, v.UnderlayType, v.Time, v.URL, v.Result, v.Cluster)
				}
				fmt.Println()
			}

			return nil
		},
	}

	c.Flags().StringVar(&jsonFile, "file", "", "model file")
	c.MarkFlagRequired("file")
	return c
}
