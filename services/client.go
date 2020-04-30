package services

import (
	"fmt"
	"time"

	huaweisdk "github.com/huaweicloud/golangsdk"
	"github.com/huaweicloud/golangsdk/openstack/cce/v3/clusters"
	"github.com/huaweicloud/golangsdk/openstack/cce/v3/nodes"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/extensions/keypairs"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/extensions/secgroups"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/extensions/servergroups"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/servers"
	"github.com/huaweicloud/golangsdk/openstack/networking/v1/eips"
	"github.com/huaweicloud/golangsdk/openstack/networking/v1/subnets"
	"github.com/huaweicloud/golangsdk/openstack/networking/v1/vpcs"

	"github.com/opentelekomcloud-infra/crutch-house/clientconfig"
)

const (
	maxAttempts   = 50
	waitInterval  = 5 * time.Second
	defaultRegion = "eu-de"
)

type Client interface {
	Authenticate() error
	InitNetwork() error
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
	addInboundRule(secGroupID string, fromPort int, toPort int) error
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
	DeleteCluster(clusterID string) error
	CreateNodes(opts *CreateNodesOpts, count int) (*nodes.Nodes, error)
	DeleteNodes(clusterID, nodeID string) error
}

// client contains service clients
type client struct {
	Provider *huaweisdk.ProviderClient

	ComputeV2 *huaweisdk.ServiceClient
	VPC       *huaweisdk.ServiceClient
	CCE       *huaweisdk.ServiceClient

	opts *clientconfig.ClientOpts
}

func NewClient(opts *clientconfig.ClientOpts) Client {
	opts.EndpointType = clientconfig.GetEndpointType(opts.EndpointType)
	if opts.RegionName == "" {
		opts.RegionName = defaultRegion
	}
	return &client{opts: opts}
}

var userAgent = fmt.Sprintf("otc-crutch-house/v0.1")

// AuthenticateWithToken authenticate client in the cloud with token (either directly or via username/password)
func (c *client) Authenticate() error {
	if c.Provider != nil {
		return nil
	}
	authClient, err := clientconfig.AuthenticatedClient(c.opts)
	if err != nil {
		return err
	}
	c.Provider = authClient
	c.Provider.UserAgent.Prepend(userAgent)
	return nil
}
