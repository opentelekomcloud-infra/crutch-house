package services

import (
	"fmt"
	"strings"
	"time"

	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
)

const (
	maxAttempts         = 50
	waitInterval        = 5 * time.Second
	defaultEndpointType = golangsdk.AvailabilityPublic

	userAgent = "otc-crutch-house/v0.1"
)

// Client contains service clients
type Client struct {
	Provider *golangsdk.ProviderClient

	ECS       *golangsdk.ServiceClient
	ComputeV2 *golangsdk.ServiceClient
	NetworkV2 *golangsdk.ServiceClient
	VPC       *golangsdk.ServiceClient
	CCE       *golangsdk.ServiceClient

	cloud *openstack.Cloud
}

func NewCloudClient(cloud *openstack.Cloud) *Client {
	return &Client{cloud: cloud}
}

func NewClient(prefix string) (*Client, error) {
	env := openstack.NewEnv(prefix)
	cloud, err := env.Cloud()
	if err != nil {
		return nil, fmt.Errorf("failed to load cloud config: %s", err)
	}
	return &Client{cloud: cloud}, nil
}

// Authenticate - authenticate client in the cloud with token (either directly or via username/password)
func (c *Client) Authenticate() error {
	if c.Provider != nil && c.Provider.Token() != "" {
		return nil
	}
	providerClient, err := openstack.AuthenticatedClientFromCloud(c.cloud)
	if err != nil {
		return err
	}
	c.Provider = providerClient
	c.Provider.UserAgent.Prepend(userAgent)
	return nil
}

func (c *Client) Token() (string, error) {
	if c.Provider == nil || c.Provider.Token() == "" {
		if err := c.Authenticate(); err != nil {
			return "", err
		}
	}
	return c.Provider.Token(), nil
}

var validEndpointTypes = []string{"public", "internal", "admin"}

// getAvailability is a helper method to determine the endpoint type
// requested by the user.
func getAvailability(endpointType string) golangsdk.Availability {
	for _, eType := range validEndpointTypes {
		if strings.HasPrefix(endpointType, eType) {
			return golangsdk.Availability(eType)
		}
	}
	return defaultEndpointType
}

// NewServiceClient is a convenience function to get a new service client.
func (c *Client) NewServiceClient(service string) (*golangsdk.ServiceClient, error) {
	if err := c.Authenticate(); err != nil {
		return nil, err
	}
	eo := golangsdk.EndpointOpts{
		Region:       c.cloud.RegionName,
		Availability: getAvailability(c.cloud.EndpointType),
	}

	switch service {
	case "ecs":
		return openstack.NewComputeV1(c.Provider, eo)
	case "compute":
		return openstack.NewComputeV2(c.Provider, eo)
	case "dns":
		return openstack.NewDNSV2(c.Provider, eo)
	case "identity":
		return openstack.NewIdentityV3(c.Provider, eo)
	case "image":
		return openstack.NewImageServiceV2(c.Provider, eo)
	case "vpc":
		return openstack.NewNetworkV1(c.Provider, eo)
	case "network":
		return openstack.NewNetworkV2(c.Provider, eo)
	case "object-store":
		return openstack.NewObjectStorageV1(c.Provider, eo)
	case "cce":
		return openstack.NewCCE(c.Provider, eo)
	case "orchestration":
		return openstack.NewOrchestrationV1(c.Provider, eo)
	case "sharev2":
		return openstack.NewSharedFileSystemV2(c.Provider, eo)
	case "volume":
		volumeVersion := "2"
		if v := c.cloud.VolumeAPIVersion; v != "" {
			volumeVersion = v
		}

		switch volumeVersion {
		case "v1", "1":
			return openstack.NewBlockStorageV1(c.Provider, eo)
		case "v2", "2":
			return openstack.NewBlockStorageV2(c.Provider, eo)
		case "v3", "3":
			return openstack.NewBlockStorageV3(c.Provider, eo)
		default:
			return nil, fmt.Errorf("invalid volume API version")
		}
	}

	return nil, fmt.Errorf("unable to create a service client for %s", service)
}
