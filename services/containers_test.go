package services

import (
	"os"
	"testing"

	"github.com/huaweicloud/golangsdk/openstack/cce/v3/clusters"
	"github.com/huaweicloud/golangsdk/openstack/cce/v3/nodes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initCCE(t *testing.T, client Client) {
	require.NoError(t, client.InitCCE())
}

func TestClient_ClusterLifecycle(t *testing.T) {
	cleanupResources(t)

	client := computeClient(t)
	initNetwork(t, client)
	initCCE(t, client)

	vpc, err := client.CreateVPC(vpcName)
	require.NoError(t, err)
	defer deleteVPC(t, vpc.ID)

	subnet, err := client.CreateSubnet(vpc.ID, subnetName)
	require.NoError(t, err)
	defer deleteSubnet(t, vpc.ID, subnet.ID)
	require.NoError(t, client.WaitForSubnetStatus(subnet.ID, "ACTIVE"))

	ip, err := client.CreateEIP(&ElasticIPOpts{})
	require.NoError(t, err)
	defer func() {
		_ = client.DeleteFloatingIP(ip.PublicAddress)
	}()

	clusterName := RandomString(10, "crutch-")
	opts := &CreateClusterOpts{
		Name:               clusterName,
		ClusterType:        ClusterTypeECS,
		FlavorID:           "cce.s1.small",
		Description:        "Test CCE cluster",
		AuthenticationMode: "rbac",
		VpcID:              vpc.ID,
		SubnetID:           subnet.ID,
		FloatingIP:         ip.PublicAddress,
		ContainerNetwork: clusters.ContainerNetworkSpec{
			Mode: ContainerNetworkModeOverlay,
		},
	}

	kp, err := client.CreateKeyPair(kpName, "")
	require.NoError(t, err)
	defer func() { _ = client.DeleteKeyPair(kpName) }()

	cluster, err := client.CreateCluster(opts)
	require.NoError(t, err)

	nodeOpts := &CreateNodesOpts{
		Name:             "node-test",
		ClusterID:        cluster.Metadata.Id,
		Region:           os.Getenv("OTC_PROJECT_NAME"),
		KeyPair:          kp.Name,
		FlavorID:         "s2.large.2",
		AvailabilityZone: "eu-de-01",
		RootVolume: nodes.VolumeSpec{
			Size:       40,
			VolumeType: "SATA",
		},
		DataVolumes: []nodes.VolumeSpec{
			{
				Size:       100,
				VolumeType: "SATA",
			},
		},
		MaxPods:   10,
		EipCount:  1,
		PublicKey: kp.PublicKey,
	}
	created, err := client.CreateNodes(nodeOpts, 2)
	require.NoError(t, err)

	assert.NoError(t, client.DeleteNodes(cluster.Metadata.Id, created.Metadata.Id))
	assert.NoError(t, client.DeleteCluster(cluster.Metadata.Id))
}
