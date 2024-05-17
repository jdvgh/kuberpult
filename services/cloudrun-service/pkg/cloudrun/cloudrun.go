/*This file is part of kuberpult.

Kuberpult is free software: you can redistribute it and/or modify
it under the terms of the Expat(MIT) License as published by
the Free Software Foundation.

Kuberpult is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
MIT License for more details.

You should have received a copy of the MIT License
along with kuberpult. If not, see <https://directory.fsf.org/wiki/License:Expat>.

Copyright freiheit.com*/

package cloudrun

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/run/v1"
)

const (
	serviceLocationLabel       = "cloud.googleapis.com/location"
	operationIdAnnotation      = "run.googleapis.com/operation-id"
	serviceReady               = "Ready"
	serviceConfigurationsReady = "ConfigurationsReady"
	serviceRoutesReady         = "RoutesReady"
)

var (
	runService *run.APIService
)

func Init(ctx context.Context) error {
	var err error
	runService, err = run.NewService(ctx)
	return err
}

func Deploy(ctx context.Context, svc *run.Service) error {
	serviceName := svc.Metadata.Name
	// Get the full path of the project. Example: projects/<project-id>/locations/<region>
	parent, err := getParent(svc)
	servicePath := fmt.Sprintf("%s/services/%s", parent, serviceName)
	if err != nil {
		return err
	}
	req := runService.Projects.Locations.Services.List(parent)
	resp, err := req.Do()
	if err != nil {
		return err
	}
	var serviceCallResp *run.Service
	// If the service is already deployed before, then we need to call ReplaceService. Otherwise, we call Create.
	if isPresent(resp.Items, serviceName) {
		serviceCall := runService.Projects.Locations.Services.ReplaceService(servicePath, svc)
		serviceCallResp, err = serviceCall.Do()
		if err != nil {
			return err
		}
	} else {
		serviceCall := runService.Projects.Locations.Services.Create(parent, svc)
		serviceCallResp, err = serviceCall.Do()
		if err != nil {
			return err
		}
	}
	if err := waitForOperation(parent, serviceCallResp, 60); err != nil {
		return err
	}
	getServiceCall := runService.Projects.Locations.Services.Get(servicePath)
	serviceResp, err := getServiceCall.Do()
	if err != nil {
		return err
	}
	condition, err := GetServiceReadyCondition(serviceResp)
	if err != nil {
		return err
	}
	if condition.Status != "True" {
		return fmt.Errorf("service not ready: %s", condition)
	}
	return nil
}

func GetServiceReadyCondition(serviceCallResponse *run.Service) (ServiceReadyCondition, error) {
	//exhaustruct:ignore
	serviceReadyCondition := ServiceReadyCondition{
		Status:   "",
		Name:     serviceCallResponse.Metadata.Name,
		Revision: serviceCallResponse.Status.ObservedGeneration,
	}
	conditions := serviceCallResponse.Status.Conditions
	for _, condition := range conditions {
		if condition.Type == serviceReady {
			serviceReadyCondition.Status = condition.Status
			serviceReadyCondition.Reason = condition.Reason
			serviceReadyCondition.Message = condition.Message
		}
	}
	if serviceReadyCondition.Status == "" {
		return serviceReadyCondition, serviceReadyConditionError{serviceCallResponse.Metadata.Name}
	}
	return serviceReadyCondition, nil
}

func getOperationId(parent string, serviceCallResp *run.Service) (string, error) {
	operationId, exists := serviceCallResp.Metadata.Annotations[operationIdAnnotation]
	if !exists {
		return "", operationIdMissingError{serviceCallResp.Metadata.Name}
	}
	return fmt.Sprintf("%s/operations/%s", parent, operationId), nil
}

func waitForOperation(parent string, serviceCallResp *run.Service, timeout time.Duration) error {
	operationId, err := getOperationId(parent, serviceCallResp)
	if err != nil {
		return err
	}
	opService := run.NewProjectsLocationsOperationsService(runService)
	//exhaustruct:ignore
	waitOperationRequest := &run.GoogleLongrunningWaitOperationRequest{
		Timeout: fmt.Sprintf("%ds", timeout),
	}
	opServiceCall := opService.Wait(operationId, waitOperationRequest)
	operationResp, err := opServiceCall.Do()
	if err != nil {
		return fmt.Errorf("failed to wait for the service %s: %s", serviceCallResp.Metadata.Name, err)
	}
	if !operationResp.Done {
		return fmt.Errorf("service %s creation exceeded the timeout of %d seconds", serviceCallResp.Metadata.Name, timeout)
	}
	return nil
}
func isPresent(services []*run.Service, serviceName string) bool {
	for _, service := range services {
		if service.Metadata.Name == serviceName {
			return true
		}
	}
	return false
}

func getParent(svc *run.Service) (string, error) {
	namespace := svc.Metadata.Namespace
	if namespace == "" {
		return "", serviceConfigError{name: svc.Metadata.Name, namespaceMissing: true, locationMissing: false}
	}
	location, exists := svc.Metadata.Labels[serviceLocationLabel]
	if !exists {
		return "", serviceConfigError{name: svc.Metadata.Name, locationMissing: true, namespaceMissing: false}
	}
	return fmt.Sprintf("projects/%s/locations/%s", namespace, location), nil
}