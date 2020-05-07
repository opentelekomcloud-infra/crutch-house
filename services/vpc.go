package services

import (
	"fmt"

	"github.com/huaweicloud/golangsdk"
	"github.com/huaweicloud/golangsdk/openstack/networking/v1/eips"
	"github.com/huaweicloud/golangsdk/openstack/networking/v1/subnets"
	"github.com/huaweicloud/golangsdk/openstack/networking/v1/vpcs"

	"github.com/opentelekomcloud-infra/crutch-house/clientconfig"
)

const (
	vpcCIDR        = "192.168.0.0/20"
	subnetCIDR     = "192.168.0.0/24"
	primaryDNS     = "100.125.4.25"
	secondaryDNS   = "8.8.8.8"
	defaultGateway = "192.168.0.1"
	bandwidthName  = "default-bandwidth"
)

var defaultDNS = []string{primaryDNS, secondaryDNS}

// InitVPC initializes VPC v1 service
func (c *client) InitVPC() error {
	if c.VPC != nil {
		return nil
	}
	nw, err := clientconfig.NewServiceClient("vpc", c.opts)
	if err != nil {
		return err
	}
	c.VPC = nw
	return nil
}

// CreateVPC creates new VPC by d.VpcName
func (c *client) CreateVPC(vpcName string) (*vpcs.Vpc, error) {
	return vpcs.Create(c.VPC, vpcs.CreateOpts{
		Name: vpcName,
		CIDR: vpcCIDR,
	}).Extract()
}

// GetVPCDetails returns details of VPC
func (c *client) GetVPCDetails(vpcID string) (*vpcs.Vpc, error) {
	return vpcs.Get(c.VPC, vpcID).Extract()
}

// FindVPC find VPC in list by its name and return VPC ID
func (c *client) FindVPC(vpcName string) (string, error) {
	opts := vpcs.ListOpts{
		Name: vpcName,
	}
	vpcList, err := vpcs.List(c.VPC, opts)
	if err != nil {
		return "", err
	}
	if len(vpcList) == 0 {
		return "", nil
	}
	if len(vpcList) > 1 {
		return "", fmt.Errorf("multiple VPC found by name %s. Please provide VPC ID instead", vpcName)
	}
	return vpcList[0].ID, nil
}

// WaitForVPCStatus waits until VPC is in given status
func (c *client) WaitForVPCStatus(vpcID, status string) error {
	return WaitForSpecificOrError(func() (b bool, err error) {
		cur, err := c.GetVPCDetails(vpcID)
		if err != nil {
			return true, err
		}
		if cur.Status == "ERROR" {
			return true, fmt.Errorf("VPC creation failed. Instance `%s` is in ERROR state", vpcID)
		}
		if cur.Status == status {
			return true, nil
		}
		return false, nil
	}, maxAttempts, waitInterval)
}

// DeleteVPC removes existing VPC
func (c *client) DeleteVPC(vpcID string) error {
	return vpcs.Delete(c.VPC, vpcID).Err
}

// CreateSubnet creates new Subnet and set Driver.SubnetID
func (c *client) CreateSubnet(vpcID string, subnetName string) (*subnets.Subnet, error) {
	return subnets.Create(c.VPC, subnets.CreateOpts{
		VPC_ID:     vpcID,
		Name:       subnetName,
		CIDR:       subnetCIDR,
		DnsList:    defaultDNS,
		GatewayIP:  defaultGateway,
		EnableDHCP: true,
	},
	).Extract()
}

// FindSubnet find subnet by name in given VPC and return ID
func (c *client) FindSubnet(vpcID string, subnetName string) (string, error) {
	subnetList, err := subnets.List(c.VPC, subnets.ListOpts{
		Name:   subnetName,
		VPC_ID: vpcID,
	})
	if err != nil {
		return "", err
	}
	if len(subnetList) == 0 {
		return "", nil
	}
	if len(subnetList) > 1 {
		return "", fmt.Errorf("multiple Subnets found by name %s in VPC %s. "+
			"Please provide Subnet ID instead", subnetName, vpcID)
	}
	return subnetList[0].ID, nil
}

// GetSubnetStatus returns details of subnet by ID
func (c *client) GetSubnetStatus(subnetID string) (*subnets.Subnet, error) {
	return subnets.Get(c.VPC, subnetID).Extract()
}

// WaitForSubnetStatus waits for subnet to be in given status
func (c *client) WaitForSubnetStatus(subnetID string, status string) error {
	return WaitForSpecificOrError(func() (b bool, err error) {
		curStatus, err := c.GetSubnetStatus(subnetID)
		if err != nil {
			return true, err
		}
		if curStatus.Status == "ERROR" {
			return true, fmt.Errorf("subnet `%s` is in error status", subnetID)
		}
		if curStatus.Status == status {
			return true, nil
		}
		return false, nil
	}, maxAttempts, waitInterval)
}

// DeleteSubnet removes subnet from VPC
func (c *client) DeleteSubnet(vpcID string, subnetID string) error {
	return subnets.Delete(c.VPC, vpcID, subnetID).Err
}

type ElasticIPOpts struct {
	IPType        string
	BandwidthSize int
	BandwidthType string
}

func (c *client) GetEIPStatus(eipID string) (string, error) {
	eip, err := eips.Get(c.VPC, eipID).Extract()
	if err != nil {
		return "", err
	}
	return eip.Status, err
}

func (c *client) CreateEIP(opts *ElasticIPOpts) (*eips.PublicIp, error) {
	if opts.IPType == "" {
		opts.IPType = "5_bgp"
	}
	if opts.BandwidthSize == 0 {
		opts.BandwidthSize = 100
	}
	if opts.BandwidthType == "" {
		opts.BandwidthType = "PER"
	}

	applyOpts := &eips.ApplyOpts{
		IP: eips.PublicIpOpts{
			Type: opts.IPType,
		},
		Bandwidth: eips.BandwidthOpts{
			Name:      bandwidthName,
			Size:      opts.BandwidthSize,
			ShareType: opts.BandwidthType,
		},
	}
	eip, err := eips.Apply(c.VPC, applyOpts).Extract()
	if err != nil {
		return nil, err
	}
	return &eip, nil
}

func (c *client) WaitForEIPActive(eipID string) error {
	return golangsdk.WaitFor(30, func() (bool, error) {
		status, err := c.GetEIPStatus(eipID)
		if err != nil {
			return true, err
		}
		if status == "ACTIVE" || status == "DOWN" {
			return true, nil
		}
		return false, nil
	})
}
