package clientconfig

import (
	"fmt"
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
	_, _, _ = getCloudPathsSetEnvVars()

	cl, err := GetCloudFromYAML(&ClientOpts{})
	require.NoError(t, err)
	require.Empty(t, cl)
}

func TestGetCloudFromPublic(t *testing.T) {
	cloudsPath, publicPath, _ := getCloudPathsSetEnvVars()

	cloudsTemplate := cloudsYamlTemplate
	cloudsPublicTemplate := cloudsPublicYamlTemplate

	f, err := os.Create(cloudsPath)
	require.NoError(t, err)

	defer f.Close()

	_, err = f.Write([]byte(cloudsTemplate))
	require.NoError(t, err)

	f, err = os.Create(publicPath)
	require.NoError(t, err)
	_, err = f.Write([]byte(cloudsPublicTemplate))
	require.NoError(t, err)

	cl, err := GetCloudFromYAML(&ClientOpts{})
	require.NoError(t, err)
	require.Contains(t, cl.AuthInfo.AuthURL, osAuthUrl2)
	require.Contains(t, cl.AuthInfo.Password, osPassword)
}

func TestGetCloudFromSecure(t *testing.T) {
	cloudsPath, _, securePath := getCloudPathsSetEnvVars()

	cloudsTemplate := cloudsYamlTemplate
	cloudsSecureTemplate := cloudsSecureYamlTemplate

	f, err := os.Create(cloudsPath)
	require.NoError(t, err)

	defer f.Close()

	_, err = f.Write([]byte(cloudsTemplate))
	require.NoError(t, err)

	f, err = os.Create(securePath)
	require.NoError(t, err)
	_, err = f.Write([]byte(cloudsSecureTemplate))
	require.NoError(t, err)

	cl, err := GetCloudFromYAML(&ClientOpts{})
	require.NoError(t, err)
	require.Contains(t, cl.AuthInfo.AuthURL, osAuthUrl)
	require.Contains(t, cl.AuthInfo.Password, osPassword2)
}

func TestGetCloudFromAllClouds(t *testing.T) {
	cloudsPath, publicPath, securePath := getCloudPathsSetEnvVars()

	cloudsTemplate := cloudsYamlTemplate
	cloudsPublicTemplate := cloudsPublicYamlTemplate
	cloudsSecureTemplate := cloudsSecureYamlTemplate

	f, err := os.Create(cloudsPath)
	require.NoError(t, err)

	defer f.Close()

	_, err = f.Write([]byte(cloudsTemplate))
	require.NoError(t, err)

	f, err = os.Create(publicPath)
	require.NoError(t, err)
	_, err = f.Write([]byte(cloudsPublicTemplate))
	require.NoError(t, err)

	f, err = os.Create(securePath)
	require.NoError(t, err)
	_, err = f.Write([]byte(cloudsSecureTemplate))
	require.NoError(t, err)

	cl, err := GetCloudFromYAML(&ClientOpts{})
	require.NoError(t, err)
	require.Contains(t, cl.AuthInfo.AuthURL, osAuthUrl2)
	require.Contains(t, cl.AuthInfo.Password, osPassword2)
}

func TestGetPureCloud(t *testing.T) {
	cloudsPath, _, _ := getCloudPathsSetEnvVars()
	cloudsTemplate := cloudsYamlTemplate

	f, err := os.Create(cloudsPath)
	require.NoError(t, err)

	defer f.Close()

	_, err = f.Write([]byte(cloudsTemplate))
	require.NoError(t, err)

	cl, err := GetCloudFromYAML(&ClientOpts{})
	require.NoError(t, err)
	require.Contains(t, cl.AuthInfo.AuthURL, osAuthUrl)
	require.Contains(t, cl.AuthInfo.Password, osPassword)
	require.Contains(t, cl.AuthInfo.Username, osUsername)
	require.Contains(t, cl.AuthInfo.ProjectName, osProjectName)
	require.Contains(t, cl.AuthInfo.UserDomainName, osDomainName)
}

func getCloudPathsSetEnvVars() (string, string, string) {
	cloudsYamlPath := fmt.Sprintf(cloudsPath, utils.RandomString(10, "clouds"))
	cloudsPublicYamlPath := fmt.Sprintf(cloudsPath, utils.RandomString(15, "clouds-public"))
	cloudsSecureYamlPath := fmt.Sprintf(cloudsPath, utils.RandomString(10, "secure"))

	_ = os.Setenv("OS_CLIENT_CONFIG_FILE", cloudsYamlPath)
	_ = os.Setenv("OS_CLIENT_VENDOR_FILE", cloudsPublicYamlPath)
	_ = os.Setenv("OS_CLIENT_SECURE_FILE", cloudsSecureYamlPath)
	_ = os.Setenv("OS_CLOUD", "test")

	return cloudsYamlPath, cloudsPublicYamlPath, cloudsSecureYamlPath
}

var cloudsYamlTemplate = fmt.Sprintf(`
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

var cloudsPublicYamlTemplate = fmt.Sprintf(`
public-clouds:
  otc:
    auth:
      auth_url: "%s"
`, osAuthUrl2)

var cloudsSecureYamlTemplate = fmt.Sprintf(`
clouds:
  test:
    auth:
      password: "%s"
`, osPassword2)
