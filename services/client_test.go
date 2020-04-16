package services

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentelekomcloud-infra/crutch-house/clientconfig"
)

const (
	authFailedMessage = "failed to authorize client"
	invalidFind       = "found %s is not what we want!"
	defaultAuthURL    = "https://iam.eu-de.otc.t-systems.com/v3"
)

var (
	vpcName    = RandomString(12, "vpc-")
	subnetName = RandomString(16, "subnet-")
	sgName     = RandomString(12, "sg-")
)

func authClient(t *testing.T) Client {
	client := NewClient(&clientconfig.ClientOpts{Cloud: "otc"})
	err := client.Authenticate()
	require.NoError(t, err, authFailedMessage)
	return client
}

func TestClient_Authenticate(t *testing.T) {
	authClient(t)
}

func TestClient_AuthenticateNoCloud(t *testing.T) {
	client := NewClient(
		&clientconfig.ClientOpts{
			RegionName:   defaultRegion,
			EndpointType: clientconfig.DefaultEndpointType,
			AuthInfo: &clientconfig.AuthInfo{
				AuthURL:     defaultAuthURL,
				Username:    os.Getenv("OTC_USERNAME"),
				Password:    os.Getenv("OTC_PASSWORD"),
				ProjectName: os.Getenv("OTC_PROJECT_NAME"),
				DomainName:  os.Getenv("OTC_DOMAIN_NAME"),
			},
		})
	err := client.Authenticate()
	require.NoError(t, err, authFailedMessage)
}

func TestClient_AuthenticateAKSK(t *testing.T) {
	client := NewClient(
		&clientconfig.ClientOpts{
			RegionName:   defaultRegion,
			EndpointType: clientconfig.DefaultEndpointType,
			AuthInfo: &clientconfig.AuthInfo{
				AuthURL:     defaultAuthURL,
				AccessKey:   os.Getenv("OTC_ACCESS_KEY_ID"),
				SecretKey:   os.Getenv("OTC_ACCESS_KEY_SECRET"),
				ProjectName: os.Getenv("OTC_PROJECT_NAME"),
				DomainName:  os.Getenv("OTC_DOMAIN_NAME"),
			},
		})
	err := client.Authenticate()
	require.NoError(t, err, authFailedMessage)
}
