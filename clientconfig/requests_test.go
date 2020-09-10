package clientconfig

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opentelekomcloud-infra/crutch-house/utils"
)

func TestGetCloudFromYAML_emptyAll(t *testing.T) {

	cloudsYamlPath := fmt.Sprintf("/tmp/%s.yaml", utils.RandomString(10, "clouds"))
	secureYamlPath := fmt.Sprintf("/tmp/%s.yaml", utils.RandomString(10, "secure"))

	_ = os.Setenv("OS_CLIENT_CONFIG_FILE", cloudsYamlPath)
	_ = os.Setenv("OS_CLIENT_SECURE_FILE", secureYamlPath)
	_ = os.Setenv("OS_CLOUD", "test-me")

	cl, err := GetCloudFromYAML(&ClientOpts{})
	require.NoError(t, err)
	require.Empty(t, cl)
}

func TestGetCloudFromPublic(t *testing.T) {
	cloudsYamlPath := fmt.Sprintf("/tmp/%s.yaml", utils.RandomString(10, "clouds"))
	cloudsPublicYamlPath := fmt.Sprintf("/tmp/%s.yaml", utils.RandomString(15, "clouds-public"))

	_ = os.Setenv("OS_CLIENT_CONFIG_FILE", cloudsYamlPath)
	_ = os.Setenv("OS_CLIENT_VENDOR_FILE", cloudsPublicYamlPath)
	_ = os.Setenv("OS_CLOUD", "test")

	var osAuthUrl = "http://url-from-clouds.yaml"
	var osProjectName = "eu-de"
	var osUsername = "otc"
	var osPassword = "Qwerty123!"
	var osDomainName = "OTC987414257102518"
	cloudsTemplate := cloudsYamlTemplate(osAuthUrl, osProjectName, osUsername, osPassword, osDomainName)

	var osAuthUrlPublic = "http://url-from-clouds-public.yaml"
	cloudsPublicTemplate := cloudsPublicYamlTemplate(osAuthUrlPublic)

	f, err := os.Create(cloudsYamlPath)
	require.NoError(t, err)

	defer f.Close()

	_, err = f.Write([]byte(cloudsTemplate))
	require.NoError(t, err)

	f, err = os.Create(cloudsPublicYamlPath)
	require.NoError(t, err)
	_, err = f.Write([]byte(cloudsPublicTemplate))
	require.NoError(t, err)

	cl, err := GetCloudFromYAML(&ClientOpts{})
	require.NoError(t, err)
	require.Contains(t, cl.AuthInfo.AuthURL, "http://url-from-clouds-public.yaml")
	require.Contains(t, cl.AuthInfo.Password, "Qwerty123!")
}

func TestGetCloudFromAllClouds(t *testing.T) {
	cloudsYamlPath := fmt.Sprintf("/tmp/%s.yaml", utils.RandomString(10, "clouds"))
	cloudsPublicYamlPath := fmt.Sprintf("/tmp/%s.yaml", utils.RandomString(15, "clouds-public"))
	cloudsSecureYamlPath := fmt.Sprintf("/tmp/%s.yaml", utils.RandomString(10, "secure"))

	_ = os.Setenv("OS_CLIENT_CONFIG_FILE", cloudsYamlPath)
	_ = os.Setenv("OS_CLIENT_VENDOR_FILE", cloudsPublicYamlPath)
	_ = os.Setenv("OS_CLIENT_Secure_FILE", cloudsSecureYamlPath)
	_ = os.Setenv("OS_CLOUD", "test")

	var osAuthUrl = "http://url-from-clouds.yaml"
	var osProjectName = "eu-de"
	var osUsername = "otc"
	var osPassword = "Qwerty123!"
	var osDomainName = "OTC987414257102518"
	cloudsTemplate := cloudsYamlTemplate(osAuthUrl, osProjectName, osUsername, osPassword, osDomainName)

	var osAuthUrlPublic = "http://url-from-clouds-public.yaml"
	cloudsPublicTemplate := cloudsPublicYamlTemplate(osAuthUrlPublic)

	var password = "SecuredPa$$w0rd1@"
	cloudsSecureTemplate := cloudsSecureYamlTemplate(password)

	f, err := os.Create(cloudsYamlPath)
	require.NoError(t, err)

	defer f.Close()

	_, err = f.Write([]byte(cloudsTemplate))
	require.NoError(t, err)

	f, err = os.Create(cloudsPublicYamlPath)
	require.NoError(t, err)
	_, err = f.Write([]byte(cloudsPublicTemplate))
	require.NoError(t, err)

	f, err = os.Create(cloudsSecureYamlPath)
	require.NoError(t, err)
	_, err = f.Write([]byte(cloudsSecureTemplate))
	require.NoError(t, err)

	cl, err := GetCloudFromYAML(&ClientOpts{})
	require.NoError(t, err)
	require.Contains(t, cl.AuthInfo.AuthURL, "http://url-from-clouds-public.yaml")
	require.Contains(t, cl.AuthInfo.Password, "SecuredPa$$w0rd1")
}

func cloudsYamlTemplate(authUrl, projectName, userName, domainName, password string) string {
	return fmt.Sprintf(`
clouds:
  test:
    profile: "otc"
    auth:
      auth_url: "%s"
      project_name: "%s"
      username: "%s"
      user_domain_name: "%s"
      password: "%s"
`, authUrl, projectName, userName, domainName, password)
}

func cloudsPublicYamlTemplate(authUrl string) string {
	return fmt.Sprintf(`
public-clouds:
  otc:
    auth:
      auth_url: "%s"
`, authUrl)
}

func cloudsSecureYamlTemplate(password string) string {
	return fmt.Sprintf(`
clouds:
  test:
    auth:
      password: "%s"
`, password)
}
