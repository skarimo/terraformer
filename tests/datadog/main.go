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

package main

import (
	"fmt"
	"github.com/GoogleCloudPlatform/terraformer/cmd"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	datadog_terraforming "github.com/GoogleCloudPlatform/terraformer/providers/datadog"
)

var (
	commandTerraformInit    = "terraform init"
	commandTerraformPlan    = "terraform plan -detailed-exitcode"
	commandTerraformDestroy = "terraform destroy -auto-approve"
	commandTerraformApply   = "terraform apply -auto-approve"
	commandTerraformOutput  = "terraform output"
	datadogResourcesPath    = "tests/datadog/resources/"
)

type DatadogConfig struct {
	apiKey string
	appKey string
}

type TerraformerDatadogConfig struct {
	filter   string
	services string
}

type Config struct {
	Datadog      DatadogConfig
	Terraformer  TerraformerDatadogConfig
	logCMDOutput bool
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Print(err)
	}
	provider := &datadog_terraforming.DatadogProvider{}
	services := getServices(cfg, provider)
	rootPath, _ := os.Getwd()

	// Switch to Datadog tests resources directory
	err = os.Chdir(datadogResourcesPath)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Initialize the Datadog provider for resource creation
	err = initializeDatadogProvider(cfg)
	if err != nil {
		log.Println("Error initializing the Datadog provider", err)
		os.Exit(1)
	}

	for _, service := range services {
		resourceIds, err := createDatadogResource(cfg, service)
		if err != nil {
			log.Printf("Error creating resource %s. Error: %s", service, err)
			os.Exit(1)
		}

		filter := fmt.Sprintf("%s=%s", service, strings.Join(resourceIds, ":"))

		log.Printf("Importing %s resource via Terraformer. IDS: %s", service, resourceIds)
		err = cmd.Import(provider, cmd.ImportOptions{
			Resources:   []string{service},
			PathPattern: cmd.DefaultPathPattern,
			PathOutput:  cmd.DefaultPathOutput,
			State:       "local",
			Connect:     true,
			Output:      "hcl",
			Filter:      []string{filter},
		}, []string{cfg.Datadog.apiKey, cfg.Datadog.appKey})
		if err != nil {
			log.Printf("Error while importing resource %s. Error: %s", service, err)
			os.Exit(1)
		}

		err = terraformPlan(cfg, provider, service)
		if err != nil {
			log.Printf("Error while runnning plan on resource %s. Error: %s", service, err)
			os.Exit(1)
		}

		// Destroy resources created
		err = destroyDatadogResources(cfg, service)
		if err != nil {
			log.Printf("Error while destroying resource %s. Error: %s", service, err)
			os.Exit(1)
		}
	}
	err = os.Chdir(rootPath)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	log.Print("Successfully created and imported resources with Terraformer")
}

func destroyDatadogResources(cfg *Config, servicePath string) error {
	// Destroy resources
	log.Printf("Destroying %s resource", servicePath)
	if err := cmdRun(cfg, []string{commandTerraformDestroy, servicePath}); err != nil {
		return err
	}

	return nil
}

func terraformPlan(cfg *Config, provider *datadog_terraforming.DatadogProvider, service string) error {
	rootPath, _ := os.Getwd()
	currentPath := cmd.Path(cmd.DefaultPathPattern, provider.GetName(), service, cmd.DefaultPathOutput)
	if err := os.Chdir(currentPath); err != nil {
		return err
	}

	log.Printf("Running terraform plan against Terraformer generated %s resource", service)
	err := cmdRun(cfg, []string{commandTerraformInit, "&&", commandTerraformPlan})
	if err != nil {
		return err
	}

	if err := os.Chdir(rootPath); err != nil {
		return err
	}

	log.Printf("terraform plan did not generate any diffs for %s", service)
	return nil
}

func getConfig() (*Config, error) {
	logCMDOutput := false
	if envVar := os.Getenv("LOG_CMD_OUTPUT"); envVar != "" {
		logCMDOutputEnv, err := strconv.ParseBool(envVar)
		if err != nil {
			return nil, err
		}
		logCMDOutput = logCMDOutputEnv
	}

	return &Config{
		Datadog: DatadogConfig{
			apiKey: os.Getenv("DD_TEST_CLIENT_API_KEY"),
			appKey: os.Getenv("DD_TEST_CLIENT_APP_KEY"),
		},
		Terraformer: TerraformerDatadogConfig{
			filter:   os.Getenv("DATADOG_TERRAFORMER_FILTER"),
			services: os.Getenv("DATADOG_TERRAFORMER_SERVICES"),
		},
		logCMDOutput: logCMDOutput,
	}, nil
}

func getServices(cfg *Config, provider *datadog_terraforming.DatadogProvider) []string {
	services := []string{}
	if cfg.Terraformer.services != "" {
		services = strings.Split(cfg.Terraformer.services, ",")
	}
	if len(services) == 0 {
		services = getAllServices(provider)
	}
	sort.Strings(services)
	return services
}

func getAllServices(provider *datadog_terraforming.DatadogProvider) []string {
	services := []string{}
	for service := range provider.GetSupportedService() {
		if service == "timeboard" {
			continue
		}
		if service == "screenboard" {
			continue
		}
		services = append(services, service)
	}
	return services
}

func createDatadogResource(cfg *Config, service string) ([]string, error) {
	// Create resource
	log.Printf("Creating %s resources", service)
	if err := cmdRun(cfg, []string{commandTerraformApply, service}); err != nil {
		return nil, err
	}

	output, err := exec.Command("sh", "-c", commandTerraformOutput).Output()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	resourceIds := parseTerraformOutput(string(output))
	log.Printf("Created %s resources. IDS: %s", service, resourceIds)

	return resourceIds, nil
}

func initializeDatadogProvider(cfg *Config) error {
	// Initialize the provider
	log.Print("Initializing the Datadog provider")
	if err := cmdRun(cfg, []string{commandTerraformInit}); err != nil {
		return err
	}
	log.Print("Successfully initialized  the Datadog provider")
	return nil
}

func cmdRun(cfg *Config, args []string) error {
	terraformApiKeyEnvVariable := fmt.Sprintf("DATADOG_API_KEY=%s", cfg.Datadog.apiKey)
	terraformAPPKeyEnvVariable := fmt.Sprintf("DATADOG_APP_KEY=%s", cfg.Datadog.appKey)

	cmd := exec.Command("sh", "-c", strings.Join(args, " "))
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, terraformApiKeyEnvVariable, terraformAPPKeyEnvVariable)
	if cfg.logCMDOutput {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err := cmd.Run()
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func parseTerraformOutput(output string) []string {
	var resourceIds []string

	outputArr := strings.Split(output, "\n")
	for _, resource := range outputArr {
		resourceArr := strings.Split(resource, " = ")
		resourceIds = append(resourceIds, resourceArr[len(resourceArr)-1])
	}
	return resourceIds
}
