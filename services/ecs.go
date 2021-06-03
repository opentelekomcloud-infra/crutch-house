package services

import (
	"fmt"

	"github.com/opentelekomcloud/gophertelekomcloud/openstack/ecs/v1/cloudservers"
)

const (
	defaultTimeout = 180
)

// InitECS initializes Compute v1 (ECS) service
func (c *Client) InitECS() error {
	if c.ECS != nil {
		return nil
	}
	cmp, err := c.NewServiceClient("ecs")
	if err != nil {
		return fmt.Errorf("failed to init ECS: %s", err)
	}
	c.ECS = cmp
	return nil
}

// CreateECSInstance - create new ECS instance
func (c *Client) CreateECSInstance(opts cloudservers.CreateOptsBuilder, timeoutSeconds int) (string, error) {
	if err := c.InitECS(); err != nil {
		return "", err
	}
	job, err := cloudservers.Create(c.ECS, opts).ExtractJobResponse()
	if err != nil {
		return "", fmt.Errorf("failed to create ECS: %s", err)
	}
	if err := cloudservers.WaitForJobSuccess(c.ECS, timeoutSeconds, job.JobID); err != nil {
		return "", fmt.Errorf("failed to wait for ECS creation success: %s", err)
	}
	entity, err := cloudservers.GetJobEntity(c.ECS, job.JobID, "server_id")
	if err != nil {
		return "", fmt.Errorf("fail to get job entity")
	}
	id, ok := entity.(string)
	if !ok {
		return "", fmt.Errorf("unexpected conversion error: can't convert ID to string")
	}
	return id, nil
}

func (c *Client) GetECSStatus(instanceID string) (*cloudservers.CloudServer, error) {
	if err := c.InitECS(); err != nil {
		return nil, err
	}
	return cloudservers.Get(c.ECS, instanceID).Extract()
}

func (c *Client) DeleteECSInstance(instanceID string) error {
	if err := c.InitECS(); err != nil {
		return err
	}
	job, err := cloudservers.Delete(c.ECS, cloudservers.DeleteOpts{
		Servers: []cloudservers.Server{
			{Id: instanceID},
		},
		DeletePublicIP: false,
		DeleteVolume:   true,
	}).ExtractJobResponse()
	if err != nil {
		return fmt.Errorf("failed to delete ECS: %s", err)
	}
	if err := cloudservers.WaitForJobSuccess(c.ECS, defaultTimeout, job.JobID); err != nil {
		return fmt.Errorf("failed to wait for ECS deletion success: %s", err)
	}
	return nil
}
