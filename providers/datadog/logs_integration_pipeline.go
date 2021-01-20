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

	datadogV1 "github.com/DataDog/datadog-api-client-go/api/v1/datadog"
	"github.com/GoogleCloudPlatform/terraformer/terraformutils"
)

var (
	// LogsIntegrationPipelineAllowEmptyValues ...
	LogsIntegrationPipelineAllowEmptyValues = []string{"support_rules"}
)

// LogsIntegrationPipelineGenerator ...
type LogsIntegrationPipelineGenerator struct {
	DatadogService
}

func (g *LogsIntegrationPipelineGenerator) createResources(logs_integration_pipelines []datadogV1.LogsPipeline) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, logs_integration_pipeline := range logs_integration_pipelines {
		// Import logs integration pipelines only
		if logs_integration_pipeline.GetIsReadOnly(){
			resourceId := logs_integration_pipeline.GetId()
			resourceName := logs_integration_pipeline.GetName()
			resources = append(resources, g.createResource(resourceId,  resourceName))
		}
	}

	return resources
}

func (g *LogsIntegrationPipelineGenerator) createResource(LogsIntegrationPipelineID string, LogsIntegrationPipelineName string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		LogsIntegrationPipelineID,
		LogsIntegrationPipelineName,
		"datadog_logs_integration_pipeline",
		"datadog",
		LogsIntegrationPipelineAllowEmptyValues,
	)
}

// InitResources Generate TerraformResources from Datadog API,
// from each integration pipeline create 1 TerraformResource.
// Need LogsPipeline ID as ID for terraform resource
func (g *LogsIntegrationPipelineGenerator) InitResources() error {
	datadogClientV1 := g.Args["datadogClientV1"].(*datadogV1.APIClient)
	authV1 := g.Args["authV1"].(context.Context)

	logs_integration_pipelines, _, err := datadogClientV1.LogsPipelinesApi.ListLogsPipelines(authV1).Execute()
	if err != nil {
		return err
	}
	g.Resources = g.createResources(logs_integration_pipelines)
	return nil
}
