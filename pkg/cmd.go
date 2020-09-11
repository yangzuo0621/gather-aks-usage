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
	c.AddCommand(createCountInfoCommand())
	c.AddCommand(createOutputCommand())
	c.AddCommand(createConvertCommand())
	return c
}

func createCountInfoCommand() *cobra.Command {
	var (
		jsonFile string
		topN     int
		datas    []BuildData
	)

	c := &cobra.Command{
		Use:          "count-info",
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
								})
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

			log.Println("Finish")

			return ioutil.WriteFile(jsonFile, b.Bytes(), 0644)
		},
	}

	c.Flags().StringVar(&jsonFile, "file", "", "model file")
	c.Flags().IntVar(&topN, "top", 0, "top N records")
	c.MarkFlagRequired("file")

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
		datas    []Data
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

				for _, v := range data.Builds {
					fmt.Printf("| %d | %s | %s | %s | %-9s |\n", v.BuildID, v.UnderlayType, v.Time, v.URL, v.Result)
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

func createConvertCommand() *cobra.Command {
	var (
		jsonFile   string
		outFile    string
		datas      []Data
		buildDatas []BuildData
	)
	c := &cobra.Command{
		Use:          "convert",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
				return err
			}
			file, _ := ioutil.ReadFile(jsonFile)
			json.Unmarshal(file, &datas)

			buildDatas = []BuildData{}
			for _, d := range datas {
				builds := []BuildInfo{}
				for _, b := range d.Builds {
					builds = append(builds, b)
				}
				buildDatas = append(buildDatas, BuildData{
					Name:       d.Name,
					PipelineID: d.PipelineID,
					Builds:     builds,
				})
			}

			b := new(bytes.Buffer)
			encoder := json.NewEncoder(b)
			encoder.SetEscapeHTML(false)
			encoder.SetIndent("", " ")
			err := encoder.Encode(buildDatas)
			if err != nil {
				return err
			}

			return ioutil.WriteFile(outFile, b.Bytes(), 0644)
		},
	}

	c.Flags().StringVar(&jsonFile, "file", "", "source file")
	c.Flags().StringVar(&outFile, "out-file", "", "destination file")
	c.MarkFlagRequired("file")
	c.MarkFlagRequired("out-file")
	return c
}
