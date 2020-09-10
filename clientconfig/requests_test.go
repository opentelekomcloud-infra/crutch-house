package clientconfig

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentelekomcloud-infra/crutch-house/utils"
)

const (
	cloudsPath = "/tmp/%s.yaml"

	osAuthUrl     = "http://url-from-clouds.yaml"
	osProjectName = "eu-de"
	osUsername    = "otc"
	osPassword    = "Qwerty123!"
	osDomainName  = "OTC987414257102518"

	osAuthUrl2  = "http://url-from-clouds-public.yaml"
	osPassword2 = "SecuredPa$$w0rd1@"
)

func TestGetCloudFromYAML_emptyAll(t *testing.T) {
	_, _, _ = prepareCloudPaths()

	cl, err := GetCloudFromYAML(&ClientOpts{})
	require.NoError(t, err)
	require.Empty(t, cl)
}

func TestGetCloudFromPublic(t *testing.T) {
	cloudsPath, publicPath, _ := prepareCloudPaths()

	f, err := os.Create(cloudsPath)
	require.NoError(t, err)

	defer f.Close()

	_, err = f.Write([]byte(cloudsYamlTemplate))
	require.NoError(t, err)

	f, err = os.Create(publicPath)
	require.NoError(t, err)
	_, err = f.Write([]byte(cloudsPublicYamlTemplate))
	require.NoError(t, err)

	cl, err := GetCloudFromYAML(&ClientOpts{})
	require.NoError(t, err)
	assert.Equal(t, cl.AuthInfo.AuthURL, osAuthUrl2)
	assert.Equal(t, cl.AuthInfo.Password, osPassword)
}

func TestGetCloudFromSecure(t *testing.T) {
	cloudsPath, _, securePath := prepareCloudPaths()

	f, err := os.Create(cloudsPath)
	require.NoError(t, err)

	defer f.Close()

	_, err = f.Write([]byte(cloudsYamlTemplate))
	require.NoError(t, err)

	f, err = os.Create(securePath)
	require.NoError(t, err)
	_, err = f.Write([]byte(cloudsSecureYamlTemplate))
	require.NoError(t, err)

	cl, err := GetCloudFromYAML(&ClientOpts{})
	require.NoError(t, err)
	assert.Equal(t, cl.AuthInfo.AuthURL, osAuthUrl)
	assert.Equal(t, cl.AuthInfo.Password, osPassword2)
}

func TestGetCloudFromAllClouds(t *testing.T) {
	cloudsPath, publicPath, securePath := prepareCloudPaths()

	f, err := os.Create(cloudsPath)
	require.NoError(t, err)

	defer f.Close()

	_, err = f.Write([]byte(cloudsYamlTemplate))
	require.NoError(t, err)

	f, err = os.Create(publicPath)
	require.NoError(t, err)
	_, err = f.Write([]byte(cloudsPublicYamlTemplate))
	require.NoError(t, err)

	f, err = os.Create(securePath)
	require.NoError(t, err)
	_, err = f.Write([]byte(cloudsSecureYamlTemplate))
	require.NoError(t, err)

	cl, err := GetCloudFromYAML(&ClientOpts{})
	require.NoError(t, err)
	assert.Equal(t, cl.AuthInfo.AuthURL, osAuthUrl2)
	assert.Equal(t, cl.AuthInfo.Password, osPassword2)
}

func TestGetPureCloud(t *testing.T) {
	cloudsPath, _, _ := prepareCloudPaths()

	f, err := os.Create(cloudsPath)
	require.NoError(t, err)

	defer f.Close()

	_, err = f.Write([]byte(cloudsYamlTemplate))
	require.NoError(t, err)

	cl, err := GetCloudFromYAML(&ClientOpts{})
	require.NoError(t, err)
	assert.Equal(t, cl.AuthInfo.AuthURL, osAuthUrl)
	assert.Equal(t, cl.AuthInfo.Password, osPassword)
	assert.Equal(t, cl.AuthInfo.Username, osUsername)
	assert.Equal(t, cl.AuthInfo.ProjectName, osProjectName)
	assert.Equal(t, cl.AuthInfo.UserDomainName, osDomainName)
}

func prepareCloudPaths() (string, string, string) {
	cloudsYamlPath := fmt.Sprintf(cloudsPath, utils.RandomString(10, "clouds"))
	cloudsPublicYamlPath := fmt.Sprintf(cloudsPath, utils.RandomString(15, "clouds-public"))
	cloudsSecureYamlPath := fmt.Sprintf(cloudsPath, utils.RandomString(10, "secure"))

	_ = os.Setenv("OS_CLIENT_CONFIG_FILE", cloudsYamlPath)
	_ = os.Setenv("OS_CLIENT_VENDOR_FILE", cloudsPublicYamlPath)
	_ = os.Setenv("OS_CLIENT_SECURE_FILE", cloudsSecureYamlPath)
	_ = os.Setenv("OS_CLOUD", "test")

	return cloudsYamlPath, cloudsPublicYamlPath, cloudsSecureYamlPath
}

var (
	cloudsYamlTemplate = fmt.Sprintf(`
clouds:
  test:
    profile: "otc"
    auth:
      auth_url: "%s"
      project_name: "%s"
      username: "%s"
      user_domain_name: "%s"
      password: "%s"
`, osAuthUrl, osProjectName, osUsername, osDomainName, osPassword)

	cloudsPublicYamlTemplate = fmt.Sprintf(`
public-clouds:
  otc:
    auth:
      auth_url: "%s"
`, osAuthUrl2)

	cloudsSecureYamlTemplate = fmt.Sprintf(`
clouds:
  test:
    auth:
      password: "%s"
`, osPassword2)
)
