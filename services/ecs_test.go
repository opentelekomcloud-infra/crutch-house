package services

import (
	"testing"

	"github.com/opentelekomcloud/gophertelekomcloud/openstack/ecs/v1/cloudservers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ecsClient(t *testing.T) Client {
	client := authClient(t)
	require.NoError(t, client.InitECS())
	return client
}

func TestClient_CreateNewECS(t *testing.T) {
	cl := ecsClient(t)

	cleanupResources(t)

	client := computeClient(t)
	initNetwork(t, client)

	grp, err := createServerGroup(client)
	require.NoError(t, err)
	defer deleteServerGroup(client, grp.ID)
	t.Log("Server group created")

	vpc, err := client.CreateVPC(vpcName)
	require.NoError(t, err)
	defer deleteVPC(t, vpc.ID)
	t.Log("VPC created")

	subnet, err := client.CreateSubnet(vpc.ID, subnetName)
	require.NoError(t, err)
	defer deleteSubnet(t, vpc.ID, subnet.ID)
	require.NoError(t, client.WaitForSubnetStatus(subnet.ID, "ACTIVE"))
	t.Log("Subnet created")

	sg, err := client.CreateSecurityGroup(sgName, PortRange{From: 22})
	require.NoError(t, err)
	defer func() { _ = client.DeleteSecurityGroup(sg.ID) }()
	t.Log("Security group created")

	kp, err := client.CreateKeyPair(kpName, "")
	require.NoError(t, err)
	defer func() { _ = client.DeleteKeyPair(kpName) }()
	t.Log("Key pair created")

	imgRef, err := client.FindImage(defaultImage)
	require.NoError(t, err)

	opts := cloudservers.CreateOpts{
		ImageRef:  imgRef,
		FlavorRef: defaultFlavor,
		Name:      "test-dmd",
		KeyName:   kp.Name,
		VpcId:     vpc.ID,
		Nics: []cloudservers.Nic{
			{SubnetId: subnet.ID},
		},
		RootVolume: cloudservers.RootVolume{
			VolumeType: "SSD",
			Size:       40,
		},
		SecurityGroups: []cloudservers.SecurityGroup{
			{ID: sg.ID},
		},
		AvailabilityZone: defaultAZ,
		ServerTags: []cloudservers.ServerTags{
			{Key: "by", Value: "dmd"},
		},
	}
	id, err := cl.CreateECSInstance(opts, defaultTimeout)
	require.NoError(t, err)
	require.NotEmpty(t, id)
	t.Log("Instance created")

	assert.NoError(t, cl.DeleteECSInstance(id))
	t.Log("Instance deleted")
}
