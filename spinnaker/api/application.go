package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"

	gate "github.com/spinnaker/spin/cmd/gateclient"
	gateapi "github.com/spinnaker/spin/gateapi"
)

func GetApplication(client *gate.GatewayClient, applicationName string, dest interface{}) error {
   app, resp, err := client.ApplicationControllerApi.GetApplicationUsingGET(client.Context, applicationName, &gateapi.ApplicationControllerApiGetApplicationUsingGETOpts{Expand: optional.NewBool(false)})
	if resp != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Application '%s' not found\n", applicationName)
		} else if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Encountered an error getting application, status code: %d\n", resp.StatusCode)
		}
	}

	if err != nil {
		return FormatAPIErrorMessage ("ApplicationControllerApi.GetApplicationUsingGET", err)
	}

	if err := mapstructure.Decode(app, dest); err != nil {
		return err
	}

	return nil
}

func CreateOrUpdateApplication(client *gate.GatewayClient, applicationName, email,
	applicationDescription string, platformHealthOnly, platformHealthOnlyShowOverride bool,
        cloud_providers []interface{}, permissions *schema.Set) error {

	jobType := "createApplication"
	jobDescription := fmt.Sprintf("Create Application: %s", applicationName)
   	app, resp, err := client.ApplicationControllerApi.GetApplicationUsingGET(client.Context, applicationName, &gateapi.ApplicationControllerApiGetApplicationUsingGETOpts{Expand: optional.NewBool(false)})
	if resp != nil && resp.StatusCode == http.StatusOK && err == nil {
		jobType = "updateApplication"
		jobDescription = fmt.Sprintf("Update Application: %s", applicationName)
	}
	
	app = map[string]interface{}{
		"instancePort":                   80,
		"name":                           applicationName,
		"email":                          email,
		"platformHealthOnly":             platformHealthOnly,
		"platformHealthOnlyShowOverride": platformHealthOnlyShowOverride,
		"description":                    applicationDescription,
	}

        if len(cloud_providers) > 0 {
		providers_csv := ""
		for i := range cloud_providers {
			if (i > 0) { providers_csv += "," }
			providers_csv += cloud_providers[i].(string)
		}
		app["cloudProviders"] = providers_csv
	}

        if permissions.Len() == 1 {
		permissions_object := make(map[string]interface{})
		list := permissions.List()
		for k, value := range list[0].(map[string]interface{}) {
			switch key := k; key {
				case "read":
					permissions_object["READ"] = value
				case "execute":
					permissions_object["EXECUTE"] = value
				case "write":
					permissions_object["WRITE"] = value
				default:
					return fmt.Errorf("invalid permissions type of %s", key)
			}
		}
		app["permissions"] = permissions_object
	}

	createAppTask := map[string]interface{}{
		"job":         []interface{}{map[string]interface{}{"type": jobType, "application": app}},
		"application": applicationName,
		"description": jobDescription,
	}

	ref, _, err := client.TaskControllerApi.TaskUsingPOST1(client.Context, createAppTask)
	if err != nil {
		return FormatAPIErrorMessage ("TaskControllerApi.TaskUsingPOST1", err)
	}

	toks := strings.Split(ref["ref"].(string), "/")
	id := toks[len(toks)-1]

	task, resp, err := client.TaskControllerApi.GetTaskUsingGET1(client.Context, id)
	attempts := 0
	for (task == nil || !taskCompleted(task)) && attempts < 5 {
		toks := strings.Split(ref["ref"].(string), "/")
		id := toks[len(toks)-1]

		task, resp, err = client.TaskControllerApi.GetTaskUsingGET1(client.Context, id)
		attempts += 1
		time.Sleep(time.Duration(attempts*attempts) * time.Second)
	}

	if err != nil {
		return FormatAPIErrorMessage ("TaskControllerApi.GetTaskUsingGET1", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("Encountered an error saving application, status code: %d\n", resp.StatusCode)
	}
	if !taskSucceeded(task) {
		return fmt.Errorf("Encountered an error saving application, task output was: %v\n", task)
	}

	return nil
}

func DeleteAppliation(client *gate.GatewayClient, applicationName string) error {
	jobSpec := map[string]interface{}{
		"type": "deleteApplication",
		"application": map[string]interface{}{
			"name": applicationName,
		},
	}

	deleteAppTask := map[string]interface{}{
		"job":         []interface{}{jobSpec},
		"application": applicationName,
		"description": fmt.Sprintf("Delete Application: %s", applicationName),
	}

	_, resp, err := client.TaskControllerApi.TaskUsingPOST1(client.Context, deleteAppTask)

	if err != nil {
		return FormatAPIErrorMessage ("TaskControllerApi.TaskUsingPOST1", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Encountered an error deleting application, status code: %d\n", resp.StatusCode)
	}

	return nil
}

func taskCompleted(task map[string]interface{}) bool {
	taskStatus, exists := task["status"]
	if !exists {
		return false
	}

	COMPLETED := [...]string{"SUCCEEDED", "STOPPED", "SKIPPED", "TERMINAL", "FAILED_CONTINUE"}
	for _, status := range COMPLETED {
		if taskStatus == status {
			return true
		}
	}
	return false
}

func taskSucceeded(task map[string]interface{}) bool {
	taskStatus, exists := task["status"]
	if !exists {
		return false
	}

	SUCCESSFUL := [...]string{"SUCCEEDED", "STOPPED", "SKIPPED"}
	for _, status := range SUCCESSFUL {
		if taskStatus == status {
			return true
		}
	}
	return false
}
