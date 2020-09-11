package services

import (
	"os"
	"testing"

	"github.com/opentelekomcloud/gophertelekomcloud/openstack/cce/v3/clusters"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/cce/v3/nodes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opentelekomcloud-infra/crutch-house/utils"
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
	defer func() { _ = client.DeleteFloatingIP(ip.PublicAddress) }()

	clusterName := utils.RandomString(10, "crutch-", "0123456789abcdefghijklmnopqrstuvwxyz")
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
	clusterID := cluster.Metadata.Id

	nodeOpts := &CreateNodesOpts{
		Name:             "node-test",
		ClusterID:        clusterID,
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
		PublicKey: kp.PublicKey,
	}
	nodeCount := 2
	created, err := client.CreateNodes(nodeOpts, nodeCount)
	require.NoError(t, err)

	status, err := client.GetNodesStatus(clusterID, created)
	assert.NoError(t, err)
	assert.Len(t, status, nodeCount)
	assert.NotContains(t, status, "")

	assert.NoError(t, client.DeleteNodes(clusterID, created))
	assert.NoError(t, client.DeleteCluster(clusterID))
}
