package services

import (
	"testing"

	"github.com/huaweicloud/golangsdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initNetwork(t *testing.T, client Client) {
	require.NoError(t, client.InitVPC())
}

func TestClient_CreateVPC(t *testing.T) {
	cleanupResources(t)
	client := authClient(t)
	initNetwork(t, client)

	vpc, err := client.CreateVPC(vpcName)
	require.NoError(t, err)

	assert.NoError(t, client.WaitForVPCStatus(vpc.ID, "OK"))

	vpcID, err := client.FindVPC(vpcName)
	assert.NoError(t, err)
	assert.Equal(t, vpc.ID, vpcID)

	assert.NoError(t, client.DeleteVPC(vpc.ID))
}

func TestClient_DeleteSecurityGroupViaVPC(t *testing.T) {
	cleanupResources(t)

	client := computeClient(t)
	initNetwork(t, client)

	vpc, err := client.CreateVPC(vpcName)
	require.NoError(t, err)
	defer deleteVPC(t, vpc.ID)

	subnet, err := client.CreateSubnet(vpc.ID, subnetName)
	require.NoError(t, err)
	defer deleteSubnet(t, vpc.ID, subnet.ID)

	sg, err := client.CreateSecurityGroup(sgName, PortRange{From: 22})
	require.NoError(t, err)
	defer func() { _ = client.DeleteSecurityGroup(sg.ID) }()

	err = client.DeleteSecurityGroupViaVPC(vpc.ID)
	assert.NoError(t, err)

	sgIDs, err := client.FindSecurityGroups([]string{sgName})
	assert.NoError(t, err)
	assert.EqualValuesf(t, sg.ID, sgIDs[0], invalidFind, "sec group")
}

func TestClient_CreateSubnet(t *testing.T) {
	cleanupResources(t)
	client := authClient(t)
	initNetwork(t, client)
	vpc, err := client.CreateVPC(vpcName)
	require.NoError(t, err)

	subnet, err := client.CreateSubnet(vpc.ID, subnetName)
	assert.NoError(t, err)

	err = client.WaitForSubnetStatus(subnet.ID, "ACTIVE")
	assert.NoError(t, err)

	found, err := client.FindSubnet(vpc.ID, subnetName)
	assert.NoError(t, err)
	assert.Equalf(t, subnet.ID, found, invalidFind, "subnet")

	assert.NoError(t, client.DeleteSubnet(vpc.ID, found))

	err = client.WaitForSubnetStatus(subnet.ID, "")
	assert.IsType(t, golangsdk.ErrDefault404{}, err)

	assert.NoError(t, client.DeleteVPC(vpc.ID))
}
