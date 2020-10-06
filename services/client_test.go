package services

import (
	"testing"

	"github.com/opentelekomcloud-infra/crutch-house/utils"
	"github.com/stretchr/testify/require"
)

const (
	authFailedMessage = "failed to authorize client"
	invalidFind       = "found %s is not what we want!"
)

var (
	vpcName    = utils.RandomString(12, "vpc-")
	subnetName = utils.RandomString(16, "subnet-")
	sgName     = utils.RandomString(12, "sg-")
)

func authClient(t *testing.T) Client {
	pref := "OS_"

	client := NewClient(pref)
	err := client.Authenticate()
	require.NoError(t, err, authFailedMessage)
	return client
}

func TestClient_Authenticate(t *testing.T) {
	authClient(t)
}
