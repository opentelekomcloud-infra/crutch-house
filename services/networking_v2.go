package services

import (
	"fmt"

	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/lbaas_v2/listeners"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/lbaas_v2/loadbalancers"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/lbaas_v2/monitors"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/lbaas_v2/pools"
)

const LBStateActive = "ACTIVE"

// InitNetworkV2 initializes OpenStack Neutron client
func (c *Client) InitNetworkV2() error {
	if c.NetworkV2 != nil {
		return nil
	}
	nw, err := c.NewServiceClient("network")
	if err != nil {
		return err
	}
	c.NetworkV2 = nw
	return nil
}

// CreateLoadBalancer creating new ELBv2
func (c *Client) CreateLoadBalancer(opts *loadbalancers.CreateOpts) (*loadbalancers.LoadBalancer, error) {
	lb, err := loadbalancers.Create(c.NetworkV2, opts).Extract()
	if err != nil {
		return nil, err
	}

	if err := c.waitForLBActive(lb.ID); err != nil {
		return lb, err
	}

	return lb, nil
}

// GetLoadBalancerDetails fetches load balancer data
func (c *Client) GetLoadBalancerDetails(id string) (*loadbalancers.LoadBalancer, error) {
	return loadbalancers.Get(c.NetworkV2, id).Extract()
}

func (c *Client) waitForLBActive(loadBalancerID string) error {
	return golangsdk.WaitFor(60, func() (bool, error) {
		lb, err := c.GetLoadBalancerDetails(loadBalancerID)
		if err != nil {
			return true, err
		}
		if lb.ProvisioningStatus == LBStateActive {
			return true, nil
		}
		return false, nil
	})
}

func (c *Client) waitForLBDeleted(loadBalancerID string) error {
	return golangsdk.WaitFor(60, func() (bool, error) {
		_, err := c.GetLoadBalancerDetails(loadBalancerID)
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

// DeleteLoadBalancer removes existing load balancer
func (c *Client) DeleteLoadBalancer(id string) error {
	if err := loadbalancers.Delete(c.NetworkV2, id).Err; err != nil {
		return err
	}
	return c.waitForLBDeleted(id)
}

// BindFloatingIPToPort binds floating IP to networking port
func (c *Client) BindFloatingIPToPort(floatingIP, portID string) error {
	page, err := floatingips.List(c.NetworkV2, floatingips.ListOpts{
		FloatingIP: floatingIP,
	}).AllPages()
	if err != nil {
		return err
	}
	ids, err := floatingips.ExtractFloatingIPs(page)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return fmt.Errorf("failed to find existing floating IP `%s`", floatingIP)
	}
	opts := floatingips.UpdateOpts{PortID: &portID}
	return floatingips.Update(c.NetworkV2, ids[0].ID, opts).Err
}

func (c *Client) CreateLBListener(opts *listeners.CreateOpts) (*listeners.Listener, error) {
	return listeners.Create(c.NetworkV2, *opts).Extract()
}

func (c *Client) DeleteLBListener(id string) error {
	return listeners.Delete(c.NetworkV2, id).Err
}

func (c *Client) CreateLBPool(opts *pools.CreateOpts) (*pools.Pool, error) {
	return pools.Create(c.NetworkV2, opts).Extract()
}

func (c *Client) DeleteLBPool(id string) error {
	return pools.Delete(c.NetworkV2, id).Err
}

func (c *Client) CreateLBMember(poolID string, opts *pools.CreateMemberOpts) (*pools.Member, error) {
	return pools.CreateMember(c.NetworkV2, poolID, *opts).Extract()
}

func (c *Client) GetLBMemberStatus(poolID, memberID string) (*pools.Member, error) {
	return pools.GetMember(c.NetworkV2, poolID, memberID).Extract()
}

func (c *Client) DeleteLBMember(poolID, memberID string) error {
	return pools.DeleteMember(c.NetworkV2, poolID, memberID).Err
}

// as it's done in terraform provider
func (c *Client) waitForLBV2viaPool(id string) error {
	pool, err := pools.Get(c.NetworkV2, id).Extract()
	if err != nil {
		return err
	}
	if pool.Loadbalancers != nil {
		// each pool has an LB in Octavia lbaasv2 API
		lbID := pool.Loadbalancers[0].ID
		return c.waitForLBActive(lbID)
	}
	if pool.Listeners != nil {
		// each pool has a listener in Neutron lbaasv2 API
		listenerID := pool.Listeners[0].ID
		listener, err := listeners.Get(c.NetworkV2, listenerID).Extract()
		if err != nil {
			return err
		}
		if listener.Loadbalancers != nil {
			lbID := listener.Loadbalancers[0].ID
			return c.waitForLBActive(lbID)
		}
	}
	return fmt.Errorf("no Load Balancer on pool %s", id)
}

func (c *Client) CreateLBMonitor(opts *monitors.CreateOpts) (*monitors.Monitor, error) {
	if err := c.waitForLBV2viaPool(opts.PoolID); err != nil {
		return nil, err
	}
	monitor, err := monitors.Create(c.NetworkV2, opts).Extract()
	if err != nil {
		return nil, err
	}
	if err := c.waitForLBV2viaPool(opts.PoolID); err != nil {
		return nil, err
	}
	return monitor, nil
}

func (c *Client) DeleteLBMonitor(id string) error {
	return monitors.Delete(c.NetworkV2, id).Err
}
