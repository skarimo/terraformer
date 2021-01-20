// Copyright 2018 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datadog

import (
	"context"
	"fmt"
	datadogV1 "github.com/DataDog/datadog-api-client-go/api/v1/datadog"
	"github.com/GoogleCloudPlatform/terraformer/terraformutils"
)

var (
	// LogsCustomPipelineAllowEmptyValues ...
	LogsCustomPipelineAllowEmptyValues = []string{"support_rules"}
)

// LogsCustomPipelineGenerator ...
type LogsCustomPipelineGenerator struct {
	DatadogService
}

func (g *LogsCustomPipelineGenerator) createResources(logs_custom_pipelines []datadogV1.LogsPipeline) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, logs_custom_pipeline := range logs_custom_pipelines {
		// Import logs custom pipelines only
		if !logs_custom_pipeline.GetIsReadOnly(){
			resourceName := logs_custom_pipeline.GetId()
			resources = append(resources, g.createResource(resourceName))
		}
	}

	return resources
}

func (g *LogsCustomPipelineGenerator) createResource(LogsCustomPipelineID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		LogsCustomPipelineID,
		fmt.Sprintf("logs_custom_pipeline_%s", LogsCustomPipelineID),
		"datadog_logs_custom_pipeline",
		"datadog",
		LogsCustomPipelineAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each custom pipeline create 1 TerraformResource.
// Need LogsPipeline ID as ID for terraform resource
func (g *LogsCustomPipelineGenerator) InitResources() error {
	datadogClientV1 := g.Args["datadogClientV1"].(*datadogV1.APIClient)
	authV1 := g.Args["authV1"].(context.Context)

	resources := []terraformutils.Resource{}
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("logs_custom_pipeline") {
			for _, value := range filter.AcceptableValues {
				logs_custom_pipeline, _, err := datadogClientV1.LogsPipelinesApi.GetLogsPipeline(authV1, value).Execute()
				if err != nil {
					return err
				}

				resources = append(resources, g.createResource(logs_custom_pipeline.GetId()))
			}
		}
	}

	if len(resources) > 0 {
		g.Resources = resources
		return nil
	}

	logs_custom_pipelines, _, err := datadogClientV1.LogsPipelinesApi.ListLogsPipelines(authV1).Execute()
	if err != nil {
		return err
	}
	g.Resources = g.createResources(logs_custom_pipelines)
	return nil
}
