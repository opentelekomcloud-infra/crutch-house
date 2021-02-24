package services

import (
	"fmt"
	"time"

	"github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/cce/v3/clusters"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/cce/v3/nodes"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/compute/v2/extensions/keypairs"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/compute/v2/extensions/secgroups"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/compute/v2/extensions/servergroups"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/compute/v2/servers"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/ecs/v1/cloudservers"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v1/eips"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v1/subnets"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v1/vpcs"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/lbaas_v2/listeners"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/lbaas_v2/loadbalancers"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/lbaas_v2/monitors"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/lbaas_v2/pools"

	"github.com/opentelekomcloud-infra/crutch-house/clientconfig"
)

const (
	maxAttempts   = 50
	waitInterval  = 5 * time.Second
	defaultRegion = "eu-de"
)

type Client interface {
	Authenticate() error
	NewServiceClient(service string) (*golangsdk.ServiceClient, error)
	Token() (string, error)
	InitVPC() error
	CreateVPC(vpcName string) (*vpcs.Vpc, error)
	GetVPCDetails(vpcID string) (*vpcs.Vpc, error)
	FindVPC(vpcName string) (string, error)
	WaitForVPCStatus(vpcID string, status string) error
	DeleteVPC(vpcID string) error
	CreateSubnet(vpcID, subnetName string) (*subnets.Subnet, error)
	FindSubnet(vpcID, subnetName string) (string, error)
	GetSubnetStatus(subnetID string) (*subnets.Subnet, error)
	WaitForSubnetStatus(subnetID string, status string) error
	DeleteSubnet(vpcID string, subnetID string) error
	GetEIPStatus(eipID string) (string, error)
	CreateEIP(opts *ElasticIPOpts) (*eips.PublicIp, error)
	WaitForEIPActive(eipID string) error
	InitCompute() error
	CreateInstance(opts *ExtendedServerOpts) (*servers.Server, error)
	StartInstance(instanceID string) error
	StopInstance(instanceID string) error
	RestartInstance(instanceID string) error
	DeleteInstance(instanceID string) error
	FindInstance(name string) (string, error)
	GetInstanceStatus(instanceID string) (*servers.Server, error)
	WaitForInstanceStatus(instanceID string, status string) error
	InstanceBindToIP(instanceID, ip string) (bool, error)
	GetPublicKey(keyPairName string) ([]byte, error)
	CreateKeyPair(name string, publicKey string) (*keypairs.KeyPair, error)
	FindKeyPair(name string) (string, error)
	DeleteKeyPair(name string) error
	FindFlavor(flavorName string) (string, error)
	FindImage(imageName string) (string, error)
	CreateSecurityGroup(securityGroupName string, ports ...PortRange) (*secgroups.SecurityGroup, error)
	FindSecurityGroups(secGroups []string) ([]string, error)
	DeleteSecurityGroup(securityGroupID string) error
	WaitForGroupDeleted(securityGroupID string) error
	BindFloatingIP(floatingIP, instanceID string) error
	UnbindFloatingIP(floatingIP, instanceID string) error
	FindFloatingIP(floatingIP string) (addressID string, err error)
	DeleteFloatingIP(floatingIP string) error
	FindServerGroup(groupName string) (result string, err error)
	AddTags(instanceID string, serverTags []string) error
	CreateServerGroup(opts *servergroups.CreateOpts) (*servergroups.ServerGroup, error)
	DeleteServerGroup(groupID string) error
	InitCCE() error
	CreateCluster(opts *CreateClusterOpts) (*clusters.Clusters, error)
	GetCluster(clusterID string) (*clusters.Clusters, error)
	GetClusterCertificate(clusterID string) (*clusters.Certificate, error)
	UpdateCluster(clusterID string, opts *clusters.UpdateSpec) error
	DeleteCluster(clusterID string) error
	CreateNodes(opts *CreateNodesOpts, count int) ([]string, error)
	GetNodesStatus(clusterID string, nodeIDs []string) ([]*nodes.Status, error)
	DeleteNodes(clusterID string, nodeIDs []string) error
	InitNetworkV2() error
	CreateLoadBalancer(opts *loadbalancers.CreateOpts) (*loadbalancers.LoadBalancer, error)
	GetLoadBalancerDetails(id string) (*loadbalancers.LoadBalancer, error)
	DeleteLoadBalancer(id string) error
	BindFloatingIPToPort(floatingIP, portID string) error
	CreateLBListener(opts *listeners.CreateOpts) (*listeners.Listener, error)
	DeleteLBListener(id string) error
	CreateLBPool(opts *pools.CreateOpts) (*pools.Pool, error)
	DeleteLBPool(id string) error
	CreateLBMember(poolID string, opts *pools.CreateMemberOpts) (*pools.Member, error)
	GetLBMemberStatus(poolID, memberID string) (*pools.Member, error)
	DeleteLBMember(poolID, memberID string) error
	CreateLBMonitor(opts *monitors.CreateOpts) (*monitors.Monitor, error)
	DeleteLBMonitor(id string) error
	InitECS() error
	CreateECSInstance(opts cloudservers.CreateOptsBuilder, timeoutSeconds int) (string, error)
	GetECSStatus(instanceID string) (*cloudservers.CloudServer, error)
	DeleteECSInstance(instanceID string) error
}

// client contains service clients
type client struct {
	Provider *golangsdk.ProviderClient

	ECS       *golangsdk.ServiceClient
	ComputeV2 *golangsdk.ServiceClient
	NetworkV2 *golangsdk.ServiceClient
	VPC       *golangsdk.ServiceClient
	CCE       *golangsdk.ServiceClient

	cloud *openstack.Cloud
}

func NewCloudClient(cloud *openstack.Cloud) Client {
	return &client{cloud: cloud}
}

func NewClient(prefix string) (Client, error) {
	env := openstack.NewEnv(prefix)
	cloud, err := env.Cloud()
	if err != nil {
		return nil, fmt.Errorf("failed to load cloud config: %s", err)
	}
	return &client{cloud: cloud}, nil
}

var userAgent = fmt.Sprintf("otc-crutch-house/v0.2.4")

// AuthenticateWithToken authenticate client in the cloud with token (either directly or via username/password)
func (c *client) Authenticate() error {
	if c.Provider != nil {
		return nil
	}
	authClient, err := openstack.AuthenticatedClientFromCloud(c.cloud)
	if err != nil {
		return err
	}
	c.Provider = authClient
	c.Provider.UserAgent.Prepend(userAgent)
	return nil
}

func (c *client) Token() (string, error) {
	if token := c.Provider.Token(); token != "" {
		return token, nil
	}

	if err := c.Authenticate(); err != nil {
		return "", err
	}

	return c.Provider.Token(), nil

}

// NewServiceClient is a convenience function to get a new service client.
func (c *client) NewServiceClient(service string) (*golangsdk.ServiceClient, error) {
	if err := c.Authenticate(); err != nil {
		return nil, err
	}
	region := c.cloud.RegionName
	if region == "" {
		region = defaultRegion
	}
	eo := golangsdk.EndpointOpts{
		Region:       region,
		Availability: golangsdk.Availability(clientconfig.GetEndpointType(c.cloud.EndpointType)),
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
