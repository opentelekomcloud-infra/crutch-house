package services

import (
	"github.com/stretchr/testify/suite"
	"os"
	"testing"

	"github.com/opentelekomcloud-infra/crutch-house/utils"
	"github.com/stretchr/testify/require"
)

const (
	authFailedMessage = "failed to authorize client"
	invalidFind       = "found %s is not what we want!"

	defaultAuthURL = "https://iam.eu-de.otc.t-systems.com/v3"

	prefNoCloud = "ANC_"
	prefAKSK    = "AKSK_"
	prefToken   = "TOK_"
)

var (
	vpcName    = utils.RandomString(12, "vpc-")
	subnetName = utils.RandomString(16, "subnet-")
	sgName     = utils.RandomString(12, "sg-")
)

// copyEnvVars returning list of set vars
func copyEnvVars(toPrefix string, vars ...string) (setVars []string) {
	_ = os.Setenv(toPrefix+"AUTH_URL", defaultAuthURL)
	for _, v := range vars {
		value := os.Getenv("OTC_" + v)
		key := toPrefix + v
		setVars = append(setVars, key)
		_ = os.Setenv(key, value)
	}
	return
}

func cleanUpEnvVars(vars []string) {
	for _, v := range vars {
		_ = os.Unsetenv(v)
	}
}

func authClient(t *testing.T) Client {
	pref := "OS_"

	client := NewClient(pref)
	err := client.Authenticate()
	require.NoError(t, err, authFailedMessage)
	return client
}

func (s *ClientTestSuite) TestClient_Authenticate() {
	authClient(s.T())
}

func (s *ClientTestSuite) TestClient_AuthenticateNoCloud() {
	pref := prefNoCloud
	client := NewClient(pref)
	err := client.Authenticate()
	require.NoError(s.T(), err, authFailedMessage, err)
}

func (s *ClientTestSuite) TestClient_AuthenticateAKSK() {
	client := NewClient(prefAKSK)
	err := client.Authenticate()
	require.NoError(s.T(), err, authFailedMessage, err)
}

func (s *ClientTestSuite) TestClient_AuthenticateToken() {
	// Use token from standard auth
	preClient := authClient(s.T())
	tok, err := preClient.Token()
	s.Require().NoError(err)
	s.Require().NoError(os.Setenv(prefToken+"TOKEN", tok))

	client := NewClient(prefToken)
	err = client.Authenticate()
	require.NoError(s.T(), err, authFailedMessage, err)
}

type ClientTestSuite struct {
	vars []string
	suite.Suite
}

func (s *ClientTestSuite) SetupSuite() {
	s.vars = make([]string, 0, 10)

	// pw, no cloud
	vars := copyEnvVars(prefNoCloud, "USERNAME", "PROJECT_NAME", "PASSWORD", "DOMAIN_NAME")
	s.vars = append(s.vars, vars...)

	// ak/sk
	vars = copyEnvVars(prefAKSK, "ACCESS_KEY_ID", "PROJECT_NAME", "ACCESS_KEY_SECRET")
	s.vars = append(s.vars, vars...)

	// token
	vars = copyEnvVars(prefToken)
	s.vars = append(s.vars, vars...)
}

func (s *ClientTestSuite) TearDownSuite() {
	cleanUpEnvVars(s.vars)
}

func TestClient_Authenticate(t *testing.T) {
	if lookUpEnvVars("OTC_", "USERNAME", "PROJECT_NAME",
		"PASSWORD", "DOMAIN_NAME", "ACCESS_KEY_ID", "PROJECT_NAME",
		"ACCESS_KEY_SECRET", "TOKEN") {
		suite.Run(t, new(ClientTestSuite))
	}
	t.Skip()
}

func lookUpEnvVars(prefix string, vars ...string) bool {
	for _, v := range vars {
		if _, flag := os.LookupEnv(prefix + v); !flag {
			return false
		}
	}
	return true
}
