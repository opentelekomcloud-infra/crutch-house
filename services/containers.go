package services

import (
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/cce/v3/clusters"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/cce/v3/nodes"

	"github.com/hashicorp/go-multierror"

	"github.com/opentelekomcloud-infra/crutch-house/clientconfig"
)

const (
	ClusterTypeECS = "VirtualMachine"
	ClusterTypeBMS = "BareMetal"

	ContainerNetworkModeOverlay  = "overlay_l2"
	ContainerNetworkModeUnderlay = "underlay_ipvlan"
	ContainerNetworkModeVPC      = "vpc-router"

	ClusterAvailable = "Available"
	NodeActive       = "Active"

	EulerOSVersion = "EulerOS 2.5"
)

type Metadata struct {
	Labels      map[string]string
	Annotations map[string]string
}

type CreateClusterOpts struct {
	Metadata
	Name               string
	Description        string
	ClusterType        string                        // required, VirtualMachine or BareMetal
	ClusterVersion     string                        // optional, uses latest available version by default
	FlavorID           string                        // required, one of CCE flavour
	VpcID              string                        // required
	SubnetID           string                        // required
	HighwaySubnetID    string                        // optional, used for BMS
	ContainerNetwork   clusters.ContainerNetworkSpec // required, `Mode` should be one of ContainerNetworkMode const
	AuthenticationMode string                        // required, recommended: rbac
	BillingMode        int
	MultiAZ            bool
	FloatingIP         string
	ExtendParam        map[string]string
}

type CreateNodesOpts struct {
	Metadata
	Name             string
	ClusterID        string             // required
	Region           string             // required, project name actually
	FlavorID         string             // required
	AvailabilityZone string             // required
	KeyPair          string             // required
	RootVolume       nodes.VolumeSpec   // required, 40G+
	DataVolumes      []nodes.VolumeSpec // at least one is required required, 100G+
	Os               string             // by default EulerOS 2.5
	MaxPods          int
	PreInstall       string
	PostInstall      string
	EipCount         int
	EipOpts          ElasticIPOpts
	BillingMode      int
	PublicKey        string
	ChargingMode     int
	PerformanceType  string
	OrderID          string
	ProductID        string
}

// InitCCE initializes CCE service
func (c *client) InitCCE() error {
	if c.CCE != nil {
		return nil
	}
	cce, err := clientconfig.NewServiceClient("cce", c.opts)
	if err != nil {
		return err
	}
	c.CCE = cce
	return nil
}

func (c *client) getClusterStatus(clusterID string) (string, error) {
	state, err := clusters.Get(c.CCE, clusterID).Extract()
	if err != nil {
		return "", err
	}
	return state.Status.Phase, nil
}

func (c *client) getNodeStatus(clusterID, nodeIDs string) (string, error) {
	state, err := nodes.Get(c.CCE, clusterID, nodeIDs).Extract()
	if err != nil {
		return "", err
	}
	return state.Status.Phase, nil
}

func (c *client) waitForCluster(clusterID string) error {
	return golangsdk.WaitFor(600, func() (b bool, err error) {
		state, err := c.getClusterStatus(clusterID)
		if err != nil {
			return true, err
		}
		if state == ClusterAvailable {
			return true, nil
		}
		return false, nil
	})
}

func (c *client) waitForClusterDelete(clusterID string) error {
	return golangsdk.WaitFor(600, func() (bool, error) {
		_, err := c.getClusterStatus(clusterID)
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

// CreateCluster create CCE cluster and wait until it is available
func (c *client) CreateCluster(opts *CreateClusterOpts) (*clusters.Clusters, error) {
	opts.ExtendParam = emptyIfNil(opts.ExtendParam)
	if opts.MultiAZ {
		opts.ExtendParam["clusterAZ"] = "multi_az"
	}
	if opts.FloatingIP != "" {
		opts.ExtendParam["clusterExternalIP"] = opts.FloatingIP
	}
	createOpts := clusters.CreateOpts{
		Kind:       "Cluster",
		ApiVersion: "v3",
		Metadata: clusters.CreateMetaData{
			Name:        opts.Name,
			Labels:      emptyIfNil(opts.Labels),
			Annotations: emptyIfNil(opts.Annotations),
		},
		Spec: clusters.Spec{
			Type:        opts.ClusterType,
			Flavor:      opts.FlavorID,
			Version:     opts.ClusterVersion,
			Description: opts.Description,
			HostNetwork: clusters.HostNetworkSpec{
				VpcId:         opts.VpcID,
				SubnetId:      opts.SubnetID,
				HighwaySubnet: opts.HighwaySubnetID,
			},

			ContainerNetwork: opts.ContainerNetwork,
			Authentication: clusters.AuthenticationSpec{
				Mode:                opts.AuthenticationMode,
				AuthenticatingProxy: make(map[string]string),
			},
			BillingMode: opts.BillingMode,
			ExtendParam: opts.ExtendParam,
		},
	}

	create, err := clusters.Create(c.CCE, createOpts).Extract()

	if err != nil {
		return nil, fmt.Errorf("error creating OpenTelekomCloud cluster: %s", err)
	}

	clusterID := create.Metadata.Id
	log.Printf("Waiting for OpenTelekomCloud CCE cluster (%s) to become available", clusterID)

	return create, c.waitForCluster(clusterID)
}

func (c *client) GetCluster(clusterID string) (*clusters.Clusters, error) {
	return clusters.Get(c.CCE, clusterID).Extract()
}

func (c *client) GetClusterCertificate(clusterID string) (*clusters.Certificate, error) {
	return clusters.GetCert(c.CCE, clusterID).Extract()
}

func (c *client) DeleteCluster(clusterID string) error {
	err := clusters.Delete(c.CCE, clusterID).Err
	if err != nil {
		return err
	}
	log.Printf("Waiting for OpenTelekomCloud CCE cluster (%s) to be deleted", clusterID)
	return c.waitForClusterDelete(clusterID)
}

func installScriptEncode(script string) string {
	if _, err := base64.StdEncoding.DecodeString(script); err != nil {
		return base64.StdEncoding.EncodeToString([]byte(script))
	}
	return script
}

func (c *client) waitForMultipleNodes(clusterID string, nodeIDs []string, predicate func(nodeStatus string, err error) (bool, error)) (err *multierror.Error) {
	var errChan = make(chan error, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		go func(node string) {
			errChan <- golangsdk.WaitFor(600, func() (bool, error) {
				nodeStatus, err := c.getNodeStatus(clusterID, node)
				return predicate(nodeStatus, err)
			})
		}(nodeID)
	}

	for range nodeIDs {
		err = multierror.Append(err, <-errChan)
	}
	return err
}

func (c *client) waitForNodesActive(clusterID string, nodeIDs []string) *multierror.Error {
	return c.waitForMultipleNodes(clusterID, nodeIDs, func(nodeStatus string, err error) (bool, error) {
		if err != nil {
			return true, err
		}
		return nodeStatus == NodeActive, nil
	})
}

func (c *client) waitForNodesDeleted(clusterID string, nodeIDs []string) *multierror.Error {
	return c.waitForMultipleNodes(clusterID, nodeIDs, func(nodeStatus string, err error) (bool, error) {
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

func emptyIfNil(src map[string]string) map[string]string {
	if src == nil {
		return make(map[string]string)
	}
	return src
}

// CreateNodes create `count` nodes and wait until they are active
func (c *client) CreateNodes(opts *CreateNodesOpts, count int) ([]string, error) {
	var base64PreInstall, base64PostInstall string
	if opts.PreInstall != "" {
		base64PreInstall = installScriptEncode(opts.PreInstall)
	}
	if opts.PostInstall != "" {
		base64PostInstall = installScriptEncode(opts.PostInstall)
	}
	if opts.Os == "" {
		opts.Os = EulerOSVersion
	}
	createOpts := nodes.CreateOpts{
		Kind:       "Node",
		ApiVersion: "v3",
		Metadata: nodes.CreateMetaData{
			Name:        opts.Name,
			Labels:      opts.Labels,
			Annotations: opts.Annotations,
		},
		Spec: nodes.Spec{
			Flavor:      opts.FlavorID,
			Az:          opts.AvailabilityZone,
			Os:          opts.Os,
			Login:       nodes.LoginSpec{SshKey: opts.KeyPair},
			RootVolume:  opts.RootVolume,
			DataVolumes: opts.DataVolumes,
			PublicIP: nodes.PublicIPSpec{
				Count: opts.EipCount,
				Eip: nodes.EipSpec{
					IpType: opts.EipOpts.IPType,
					Bandwidth: nodes.BandwidthOpts{
						Size:      opts.EipOpts.BandwidthSize,
						ShareType: opts.EipOpts.BandwidthType,
					},
				},
			},
			BillingMode: opts.BillingMode,
			Count:       count,
			ExtendParam: nodes.ExtendParam{
				ChargingMode:       opts.ChargingMode,
				EcsPerformanceType: opts.PerformanceType,
				MaxPods:            opts.MaxPods,
				OrderID:            opts.OrderID,
				ProductID:          opts.ProductID,
				PublicKey:          opts.PublicKey,
				PreInstall:         base64PreInstall,
				PostInstall:        base64PostInstall,
			},
		},
	}

	clusterID := opts.ClusterID
	if err := c.waitForCluster(clusterID); err != nil {
		return nil, err
	}
	created, err := nodes.Create(c.CCE, clusterID, createOpts).Extract()
	if err != nil {
		return nil, err
	}
	nodeIDs := created.Metadata.Id
	nodeIDs = nodeIDs[:len(created.Metadata.Id)]
	nodeIDSlice := strings.Split(nodeIDs, ",")
	log.Printf("Waiting for OpenTelekomCloud CCE nodes (%s) to become available", nodeIDs)
	err = c.waitForNodesActive(clusterID, nodeIDSlice).ErrorOrNil()
	return nodeIDSlice, err
}

// GetNodesStatus returns statuses of given nodes
func (c *client) GetNodesStatus(clusterID string, nodeIDs []string) ([]*nodes.Status, error) {
	nodesChan := make(chan *nodes.Status, len(nodeIDs))
	errChan := make(chan error, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		go func(id string) {
			node, err := nodes.Get(c.CCE, clusterID, id).Extract()
			if err != nil {
				errChan <- err
				nodesChan <- nil
				return
			}
			errChan <- nil
			nodesChan <- &node.Status
		}(nodeID)
	}
	result := make([]*nodes.Status, len(nodeIDs))
	mErr := &multierror.Error{}
	for i := range nodeIDs {
		mErr = multierror.Append(mErr, <-errChan)
		result[i] = <-nodesChan
	}
	return result, mErr.ErrorOrNil()
}

// Delete all given nodes
func (c *client) DeleteNodes(clusterID string, nodeIDs []string) error {
	var errChan = make(chan error, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		go func(node string) {
			errChan <- nodes.Delete(c.CCE, clusterID, node).Err
		}(nodeID)
	}
	var err *multierror.Error
	for range nodeIDs {
		err = multierror.Append(err, <-errChan)
	}
	log.Printf("Waiting for OpenTelekomCloud CCE nodes (%s) to be deleted", strings.Join(nodeIDs, ","))
	err = multierror.Append(err, c.waitForNodesDeleted(clusterID, nodeIDs))
	return err.ErrorOrNil()
}

// Update cluster description
func (c *client) UpdateCluster(clusterID string, opts *clusters.UpdateSpec) error {
	return clusters.Update(c.CCE, clusterID, clusters.UpdateOpts{Spec: *opts}).Err
}
