package services

import (
	"fmt"
	"net"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/huaweicloud/golangsdk"
	"github.com/huaweicloud/golangsdk/openstack/compute/v2/servers"
	"github.com/huaweicloud/golangsdk/openstack/networking/v2/extensions/lbaas_v2/listeners"
	"github.com/huaweicloud/golangsdk/openstack/networking/v2/extensions/lbaas_v2/loadbalancers"
	"github.com/huaweicloud/golangsdk/openstack/networking/v2/extensions/lbaas_v2/monitors"
	"github.com/huaweicloud/golangsdk/openstack/networking/v2/extensions/lbaas_v2/pools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/getlantern/deepcopy"
)

func initClients(t *testing.T, client Client) {
	require.NoError(t, client.InitVPC())
	require.NoError(t, client.InitNetworkV2())
	require.NoError(t, client.InitCompute())
	require.NoError(t, client.InitCCE())
}

const protocol = "HTTP"

func createNodes(cl Client, opts *ExtendedServerOpts, netCIDR string, count int) (nodes []*servers.Server, addresses []string, err error) {
	nodeChan := make(chan *servers.Server, count)
	errChan := make(chan error, count)

	_, nw, err := net.ParseCIDR(netCIDR)
	if err != nil {
		return nil, nil, err
	}

	addresses = make([]string, count)
	for i := 0; i < count; i++ {
		ip, err := cidr.Host(nw, i+2)
		if err != nil {
			return nil, nil, err
		}
		address := ip.String()
		addresses[i] = address
		go func(addr, name string) {
			var o = &ExtendedServerOpts{}
			err := deepcopy.Copy(o, opts)
			if err != nil {
				nodeChan <- nil
				errChan <- err
				return
			}
			o.FixedIP = addr
			o.Name = name
			server, err := cl.CreateInstance(opts)
			nodeChan <- server
			errChan <- err
		}(address, fmt.Sprintf("%s_%d", opts.Name, i))
	}

	mErr := &multierror.Error{}
	nodes = make([]*servers.Server, count)
	for i := range nodes {
		nodes[i] = <-nodeChan
		mErr = multierror.Append(mErr, <-errChan)
	}

	return nodes, addresses, mErr.ErrorOrNil()
}

func deleteNodes(cl Client, nodes []*servers.Server) error {
	errChan := make(chan error, len(nodes))

	for _, node := range nodes {
		if node == nil {
			errChan <- nil
			continue
		}
		go func(id string) {
			err := cl.DeleteInstance(id)
			if err != nil {
				errChan <- err
				return
			}
			err = cl.WaitForInstanceStatus(id, "")

			if err == nil {
				errChan <- nil
				return
			}
			switch err.(type) {
			case golangsdk.ErrDefault404:
				errChan <- nil
			default:
				errChan <- err
			}
		}(node.ID)
	}

	err := &multierror.Error{}
	for range nodes {
		err = multierror.Append(err, <-errChan)
	}

	return err.ErrorOrNil()
}

// in-place update of servers status
func updateNodesStatus(cl Client, nodes []*servers.Server) error {
	for i, node := range nodes {
		nod, err := cl.GetInstanceStatus(node.ID)
		if err != nil {
			return err
		}
		nodes[i] = nod
	}
	return nil
}

func createMembers(cl Client, poolID string, opts *pools.CreateMemberOpts, nodes []*servers.Server, addresses []string) ([]*pools.Member, error) {
	memes := make([]*pools.Member, len(nodes))
	errChan := make(chan error, len(nodes))
	memChan := make(chan *pools.Member, len(nodes))
	for i, node := range nodes {
		if node == nil {
			memChan <- nil
			errChan <- nil
			continue
		}
		go func(opts pools.CreateMemberOpts, address string) {
			opts.Address = address
			mem, err := cl.CreateLBMember(poolID, &opts)
			memChan <- mem
			errChan <- err
		}(*opts, addresses[i])
	}

	err := &multierror.Error{}
	for i := range nodes {
		err = multierror.Append(err, <-errChan)
		memes[i] = <-memChan
	}
	return memes, err.ErrorOrNil()
}

func deleteMembers(cl Client, poolID string, memes []*pools.Member) error {
	errChan := make(chan error, len(memes))
	for _, mem := range memes {
		go func(id string) {
			err := cl.DeleteLBMember(poolID, id)
			errChan <- err
		}(mem.ID)
	}

	err := &multierror.Error{}
	for range memes {
		err = multierror.Append(err, <-errChan)
	}
	return err.ErrorOrNil()
}

func TestClient_LoadBalancerLifecycle(t *testing.T) {
	cleanupResources(t)
	client := authClient(t)
	initClients(t, client)

	vpc, err := client.CreateVPC(vpcName)
	require.NoError(t, err)
	defer deleteVPC(t, vpc.ID)
	require.NoError(t, client.WaitForVPCStatus(vpc.ID, "OK"))

	subnet, err := client.CreateSubnet(vpc.ID, subnetName)
	require.NoError(t, err)
	defer deleteSubnet(t, vpc.ID, subnet.ID)
	require.NoError(t, client.WaitForSubnetStatus(subnet.ID, "ACTIVE"))

	kp, err := client.CreateKeyPair(kpName, "")
	require.NoError(t, err)
	defer func() { _ = client.DeleteKeyPair(kpName) }()

	imgRef, err := client.FindImage(defaultImage)
	require.NoError(t, err)

	opts := &ExtendedServerOpts{
		CreateOpts: &servers.CreateOpts{
			Name:             serverName,
			FlavorName:       defaultFlavor,
			AvailabilityZone: "eu-de-01",
			Networks:         []servers.Network{{UUID: subnet.ID}},
		},
		SubnetID:    subnet.ID,
		KeyPairName: kp.Name,
		DiskOpts:    &DiskOpts{SourceID: imgRef, Size: 10, Type: "SATA"},
	}

	nodes, addresses, err := createNodes(client, opts, subnet.CIDR, 3)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, deleteNodes(client, nodes))
	}()

	eip, err := client.CreateEIP(eipOptions)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, client.DeleteFloatingIP(eip.PublicAddress))
	}()

	lb, err := client.CreateLoadBalancer(&loadbalancers.CreateOpts{
		Name:         "test-lb",
		Description:  "test lb",
		VipSubnetID:  subnet.SubnetId,
		AdminStateUp: golangsdk.Enabled,
	})
	require.NoError(t, err)
	defer func() { assert.NoError(t, client.DeleteLoadBalancer(lb.ID)) }()

	assert.NoError(t, client.BindFloatingIPToPort(eip.PublicAddress, lb.VipPortID))

	protocolPort := 80

	listener, err := client.CreateLBListener(&listeners.CreateOpts{
		LoadbalancerID: lb.ID,
		Protocol:       protocol,
		ProtocolPort:   protocolPort,
		Description:    "test listener",
	})
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, client.DeleteLBListener(listener.ID))
	}()

	pool, err := client.CreateLBPool(&pools.CreateOpts{
		LBMethod:    "LEAST_CONNECTIONS",
		Protocol:    protocol,
		Description: "test pool",
		ListenerID:  listener.ID,
	})
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, client.DeleteLBPool(pool.ID))
	}()

	require.NoError(t, updateNodesStatus(client, nodes))

	memes, err := createMembers(
		client,
		pool.ID,
		&pools.CreateMemberOpts{
			ProtocolPort: protocolPort,
			SubnetID:     subnet.SubnetId,
		},
		nodes,
		addresses,
	)
	require.NoError(t, err)
	require.NotEmpty(t, memes)
	defer func() {
		assert.NoError(t, deleteMembers(client, pool.ID, memes))
	}()

	monitor, err := client.CreateLBMonitor(&monitors.CreateOpts{
		PoolID:     pool.ID,
		Type:       "TCP",
		Delay:      10,
		Timeout:    2,
		MaxRetries: 3,
	})
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, client.DeleteLBMonitor(monitor.ID))
	}()

}
