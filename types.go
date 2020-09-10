package main

type Data struct {
	Name       string            `json:"name"`
	PipelineID int               `json:"pipeline_id"`
	Builds     map[int]BuildInfo `json:"builds,omitempty"`
}

type BuildInfo struct {
	BuildID      int    `json:"build_id"`
	UnderlayType string `json:"underlay_type"`
	Result       string `json:"result"`
	Time         string `json:"time"`
	URL          string `json:"url"`
}
