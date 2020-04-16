package services

import (
	"fmt"
	"strings"

	"github.com/huaweicloud/golangsdk"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/extensions/floatingips"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/extensions/keypairs"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/extensions/schedulerhints"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/extensions/secgroups"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/extensions/servergroups"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/extensions/startstop"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/extensions/tags"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/flavors"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/servers"
	"github.com/huaweicloud/golangsdk/openstack/imageservice/v2/images"
	"github.com/huaweicloud/golangsdk/pagination"

	"github.com/opentelekomcloud-infra/crutch-house/clientconfig"
)

// Instance statuses
const (
	InstanceStatusStopped = "SHUTOFF"
	InstanceStatusRunning = "ACTIVE"
)

// InitCompute initializes Compute v2 service
func (c *client) InitCompute() error {
	if c.ComputeV2 != nil {
		return nil
	}
	cmp, err := clientconfig.NewServiceClient("compute", c.opts)
	if err != nil {
		return err
	}
	c.ComputeV2 = cmp
	return nil
}

// DiskOpts contains source, size and type of disk
type DiskOpts struct {
	SourceID string
	Size     int
	Type     string
}

func blockDeviceOpts(opts *DiskOpts) bootfromvolume.BlockDevice {
	return bootfromvolume.BlockDevice{
		UUID:                opts.SourceID,
		VolumeSize:          opts.Size,
		VolumeType:          opts.Type,
		DeleteOnTermination: true,
		DestinationType:     "volume",
		SourceType:          "image",
	}
}

type ExtendedServerOpts struct {
	*servers.CreateOpts
	SubnetID      string
	KeyPairName   string
	DiskOpts      *DiskOpts
	ServerGroupID string
}

// CreateInstance creates new ECS
func (c *client) CreateInstance(opts *ExtendedServerOpts) (*servers.Server, error) {

	var createOpts servers.CreateOptsBuilder = &servers.CreateOpts{
		Name:             opts.Name,
		FlavorRef:        opts.FlavorRef,
		FlavorName:       opts.FlavorName,
		SecurityGroups:   opts.SecurityGroups,
		UserData:         opts.UserData,
		AvailabilityZone: opts.AvailabilityZone,
		Networks:         []servers.Network{{UUID: opts.SubnetID}},
		ServiceClient:    c.ComputeV2,
	}

	if opts.ServerGroupID != "" {
		createOpts = &schedulerhints.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			SchedulerHints: schedulerhints.SchedulerHints{
				Group: opts.ServerGroupID,
			},
		}
	}

	createOpts = &keypairs.CreateOptsExt{
		CreateOptsBuilder: createOpts,
		KeyName:           opts.KeyPairName,
	}

	blockDevice := blockDeviceOpts(opts.DiskOpts)

	createOpts = &bootfromvolume.CreateOptsExt{
		CreateOptsBuilder: createOpts,
		BlockDevice:       []bootfromvolume.BlockDevice{blockDevice},
	}

	server, err := bootfromvolume.Create(c.ComputeV2, createOpts).Extract()
	if err != nil {
		return nil, fmt.Errorf("error creating OpenTelekomCloud server: %s", err)
	}
	return server, nil
}

// StartInstance starts existing ECS instance
func (c *client) StartInstance(instanceID string) error {
	return startstop.Start(c.ComputeV2, instanceID).Err
}

// StopInstance stops existing ECS instance
func (c *client) StopInstance(instanceID string) error {
	return startstop.Stop(c.ComputeV2, instanceID).Err
}

// RestartInstance restarts ECS instance
func (c *client) RestartInstance(instanceID string) error {
	opts := &servers.RebootOpts{Type: servers.SoftReboot}
	return servers.Reboot(c.ComputeV2, instanceID, opts).Err
}

// DeleteInstance removes existing ECS instance
func (c *client) DeleteInstance(instanceID string) error {
	return servers.Delete(c.ComputeV2, instanceID).Err
}

// FindInstance returns instance ID by instance Name
func (c *client) FindInstance(name string) (string, error) {
	listOpts := servers.ListOpts{Name: name}
	pager := servers.List(c.ComputeV2, listOpts)
	serverID := ""
	err := pager.EachPage(func(page pagination.Page) (b bool, err error) {
		servs, err := servers.ExtractServers(page)
		if err != nil {
			return false, err
		}
		for _, srv := range servs {
			serverID = srv.ID
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", err
	}
	return serverID, nil
}

// GetInstanceStatus returns instance details by instance ID
func (c *client) GetInstanceStatus(instanceID string) (*servers.Server, error) {
	return servers.Get(c.ComputeV2, instanceID).Extract()
}

// WaitForInstanceStatus waits for instance to be in given status
func (c *client) WaitForInstanceStatus(instanceID string, status string) error {
	return servers.WaitForStatus(c.ComputeV2, instanceID, status, 300)
}

// InstanceBindToIP checks if instance has IP bind
func (c *client) InstanceBindToIP(instanceID string, ip string) (bool, error) {
	instanceDetails, err := c.GetInstanceStatus(instanceID)
	if err != nil {
		return false, err
	}
	for _, addrPool := range instanceDetails.Addresses {
		for _, addrDetails := range addrPool.([]interface{}) {
			details := addrDetails.(map[string]interface{})
			if details["addr"] == ip {
				return true, nil
			}
		}
	}
	return false, nil
}

// GetPublicKey returns public key data from keypair
func (c *client) GetPublicKey(keyPairName string) ([]byte, error) {
	keyPair, err := keypairs.Get(c.ComputeV2, keyPairName).Extract()
	if err != nil {
		return nil, err
	}
	return []byte(keyPair.PublicKey), nil
}

// CreateKeyPair creates new key pair from given public key string
func (c *client) CreateKeyPair(name string, publicKey string) (*keypairs.KeyPair, error) {
	opts := keypairs.CreateOpts{
		Name:      name,
		PublicKey: publicKey,
	}
	keyPair, err := keypairs.Create(c.ComputeV2, opts).Extract()
	if err != nil {
		return nil, err
	}
	return keyPair, nil
}

// FindKeyPair searches for key pair and returns public key
func (c *client) FindKeyPair(name string) (string, error) {
	pager := keypairs.List(c.ComputeV2)
	publicKey := ""
	err := pager.EachPage(func(page pagination.Page) (b bool, err error) {
		keys, err := keypairs.ExtractKeyPairs(page)
		if err != nil {
			return false, err
		}
		for _, k := range keys {
			if k.Name == name {
				publicKey = k.PublicKey
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		return "", err
	}
	return publicKey, nil
}

// DeleteKeyPair removes existing key pair
func (c *client) DeleteKeyPair(name string) error {
	return keypairs.Delete(c.ComputeV2, name).Err
}

// FindFlavor resolves `Flavor ID` for given `Flavor Name`
func (c *client) FindFlavor(flavorName string) (string, error) {
	pagedFlavors := flavors.ListDetail(c.ComputeV2, nil)
	flavorID := ""
	err := pagedFlavors.EachPage(func(page pagination.Page) (b bool, err error) {
		flavorList, err := flavors.ExtractFlavors(page)
		if err != nil {
			return false, err
		}
		for _, flav := range flavorList {
			if flav.Name == flavorName {
				flavorID = flav.ID
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		return "", err
	}
	return flavorID, nil
}

// FindImage resolve image ID by given image Name
func (c *client) FindImage(imageName string) (string, error) {
	opts := images.ListOpts{Name: imageName}
	pager := images.List(c.ComputeV2, opts)
	imageID := ""
	err := pager.EachPage(func(page pagination.Page) (b bool, err error) {
		imageList, err := images.ExtractImages(page)
		if err != nil {
			return false, err
		}
		for _, image := range imageList {
			if image.Name == imageName {
				imageID = image.ID
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		return "", err
	}
	return imageID, nil
}

const (
	cidrAll     = "0.0.0.0/0"
	tcpProtocol = "TCP"
)

func (c *client) addInboundRule(secGroupID string, fromPort int, toPort int) error {

	ruleOpts := secgroups.CreateRuleOpts{
		ParentGroupID: secGroupID,
		FromPort:      fromPort,
		ToPort:        toPort,
		CIDR:          cidrAll,
		IPProtocol:    tcpProtocol,
	}
	return secgroups.CreateRule(c.ComputeV2, ruleOpts).Err
}

// PortRange is simple sec rule port range container
type PortRange struct {
	From int
	To   int
}

// CreateSecurityGroup creates new sec group and returns group ID
func (c *client) CreateSecurityGroup(securityGroupName string, ports ...PortRange) (*secgroups.SecurityGroup, error) {
	opts := secgroups.CreateOpts{
		Name:        securityGroupName,
		Description: "Automatically created by docker-machine for OTC",
	}
	sg, err := secgroups.Create(c.ComputeV2, opts).Extract()
	if err != nil {
		return nil, err
	}
	for _, port := range ports {
		if port.To == 0 {
			port.To = port.From
		}
		if err := c.addInboundRule(sg.ID, port.From, port.To); err != nil {
			return nil, err
		}
	}
	return sg, nil
}

// found seg groups removed from source slice returning (found, missing, error)
func findSGInPagerByNameOrID(secGroups []string, pager pagination.Pager) ([]string, []string, error) {
	var secGroupIDs []string
	page, err := pager.AllPages()
	if err != nil {
		return nil, nil, err
	}
	groups, err := secgroups.ExtractSecurityGroups(page)
	if err != nil {
		return nil, nil, err
	}
	for _, found := range groups {
		idx := -1
		for i, grp := range secGroups {
			if grp == found.ID || grp == found.Name {
				idx = i
				break
			}
		}
		if idx >= 0 {
			secGroups = append(secGroups[:idx], secGroups[idx+1:]...)
			secGroupIDs = append(secGroupIDs, found.ID)
		}
	}
	return secGroupIDs, secGroups, nil
}

// FindSecurityGroups get slice of security group IDs from given security group names
func (c *client) FindSecurityGroups(secGroups []string) ([]string, error) {
	pager := secgroups.List(c.ComputeV2)
	secGroupIDs, missing, err := findSGInPagerByNameOrID(secGroups, pager)
	if err != nil {
		return nil, err
	}
	if len(missing) > 0 {
		groupsMess := strings.Join(missing, ", ")
		return secGroupIDs, fmt.Errorf("some security groups failed to be found: %v", groupsMess)
	}
	return secGroupIDs, nil
}

// DeleteSecurityGroup deletes managed security group
func (c *client) DeleteSecurityGroup(securityGroupID string) error {
	return secgroups.Delete(c.ComputeV2, securityGroupID).Err
}

// WaitForGroupDeleted polls sec group until it returns 404
func (c *client) WaitForGroupDeleted(securityGroupID string) error {
	return golangsdk.WaitFor(60, func() (b bool, e error) {
		err := secgroups.Get(c.ComputeV2, securityGroupID).Err
		if err == nil {
			return false, nil
		}
		switch err.(type) {
		case golangsdk.ErrDefault404:
			return true, nil
		default:
			return true, err
		}
	})
}

// BindFloatingIP binds floating IP to instance
func (c *client) BindFloatingIP(floatingIP string, instanceID string) error {
	opts := floatingips.AssociateOpts{FloatingIP: floatingIP}
	return floatingips.AssociateInstance(c.ComputeV2, instanceID, opts).Err
}

// UnbindFloatingIP unbinds floating IP to instance
func (c *client) UnbindFloatingIP(floatingIP string, instanceID string) error {
	opts := floatingips.DisassociateOpts{FloatingIP: floatingIP}
	return floatingips.DisassociateInstance(c.ComputeV2, instanceID, opts).Err
}

// FindFloatingIP finds given floating IP and returns ID
func (c *client) FindFloatingIP(floatingIP string) (addressID string, err error) {
	pager := floatingips.List(c.ComputeV2)
	addressID = ""
	err = pager.EachPage(func(page pagination.Page) (b bool, err error) {
		addressList, err := floatingips.ExtractFloatingIPs(page)
		if err != nil {
			return false, err
		}
		for _, ad := range addressList {
			if ad.IP == floatingIP {
				addressID = ad.ID
				return false, nil
			}
		}
		return true, nil
	})
	return
}

// DeleteFloatingIP releases floating IP
func (c *client) DeleteFloatingIP(floatingIP string) error {
	address, err := c.FindFloatingIP(floatingIP)
	if err != nil {
		return err
	}
	return floatingips.Delete(c.ComputeV2, address).Err
}

func (c *client) FindServerGroup(groupName string) (result string, err error) {
	pager := servergroups.List(c.ComputeV2)
	result = ""
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		groups, err := servergroups.ExtractServerGroups(page)
		if err != nil {
			return false, err
		}
		for _, group := range groups {
			if group.Name == groupName {
				result = group.ID
				return false, nil
			}
		}
		return true, nil
	})
	return
}

func (c *client) AddTags(instanceID string, serverTags []string) error {
	opts := tags.CreateOpts{Tags: serverTags}
	return tags.Create(c.ComputeV2, instanceID, opts).Err
}

func (c *client) CreateServerGroup(opts *servergroups.CreateOpts) (*servergroups.ServerGroup, error) {
	return servergroups.Create(c.ComputeV2, opts).Extract()
}

func (c *client) DeleteServerGroup(id string) error {
	return servergroups.Delete(c.ComputeV2, id).Err
}
