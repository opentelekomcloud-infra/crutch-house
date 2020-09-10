package clientconfig

import (
	"fmt"
	"github.com/flosch/pongo2"
	"github.com/stretchr/testify/assert"
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
	cloudsYamlPath := fmt.Sprintf("resources/%s.yaml", utils.RandomString(10, "clouds"))
	cloudsPublicYamlPath := fmt.Sprintf("resources/%s.yaml", utils.RandomString(15, "clouds-public"))

	_ = os.Setenv("OS_CLIENT_CONFIG_FILE", cloudsYamlPath)
	_ = os.Setenv("OS_CLIENT_VENDOR_FILE", cloudsPublicYamlPath)
	_ = os.Setenv("OS_CLOUD", "test")

	cloudsTemplate, err := pongo2.FromFile("resources/clouds.yaml.j2")
	require.NoError(t, err)

	cloudsPublicTemplate, err := pongo2.FromFile("resources/clouds-public.yaml.j2")
	require.NoError(t, err)

	out, err := cloudsTemplate.Execute(pongo2.Context{
		"os_auth_url":     "http://url-from-clouds.yaml",
		"os_project_name": "eu-de",
		"os_username":     "otc",
		"os_password":     "Qwerty123!",
		"os_domain_name":  "OTC987414257102518"})
	require.NoError(t, err)

	f, err := os.Create(cloudsYamlPath)
	require.NoError(t, err)

	defer f.Close()

	_, err = f.Write([]byte(out))
	require.NoError(t, err)

	out, err = cloudsPublicTemplate.Execute(pongo2.Context{
		"os_auth_url": "http://url-from-clouds-public.yaml",
	})
	require.NoError(t, err)

	f, err = os.Create(cloudsPublicYamlPath)
	require.NoError(t, err)

	_, err = f.Write([]byte(out))
	require.NoError(t, err)

	assert.NoError(t, f.Close())

	cl, err := GetCloudFromYAML(&ClientOpts{})
	require.NoError(t, err)
	require.Contains(t, cl.AuthInfo.AuthURL, "http://url-from-clouds-public.yaml")
	require.Contains(t, cl.AuthInfo.Password, "Qwerty123!")

	//err = os.Remove(cloudsYamlPath)
	//require.NoError(t, err)
	//err = os.Remove(cloudsPublicYamlPath)
	//require.NoError(t, err)
}

func TestGetCloudFromAllClouds(t *testing.T) {
	cloudsYamlPath := fmt.Sprintf("resources/%s.yaml", utils.RandomString(10, "clouds"))
	cloudsPublicYamlPath := fmt.Sprintf("resources/%s.yaml", utils.RandomString(15, "clouds-public"))
	cloudsSecureYamlPath := fmt.Sprintf("resources/%s.yaml", utils.RandomString(10, "secure"))

	_ = os.Setenv("OS_CLIENT_CONFIG_FILE", cloudsYamlPath)
	_ = os.Setenv("OS_CLIENT_VENDOR_FILE", cloudsPublicYamlPath)
	_ = os.Setenv("OS_CLIENT_Secure_FILE", cloudsSecureYamlPath)
	_ = os.Setenv("OS_CLOUD", "test")

	cloudsTemplate, err := pongo2.FromFile("resources/clouds.yaml.j2")
	require.NoError(t, err)

	cloudsPublicTemplate, err := pongo2.FromFile("resources/clouds-public.yaml.j2")
	require.NoError(t, err)

	cloudsSecureTemplate, err := pongo2.FromFile("resources/secure.yaml.j2")
	require.NoError(t, err)

	out, err := cloudsTemplate.Execute(pongo2.Context{
		"os_auth_url":     "http://url-from-clouds.yaml",
		"os_project_name": "eu-de",
		"os_username":     "otc",
		"os_password":     "Qwerty123!",
		"os_domain_name":  "OTC987414257102518"})
	require.NoError(t, err)

	f, err := os.Create(cloudsYamlPath)
	require.NoError(t, err)

	defer f.Close()

	_, err = f.Write([]byte(out))
	require.NoError(t, err)

	out, err = cloudsPublicTemplate.Execute(pongo2.Context{
		"os_auth_url": "http://url-from-clouds-public.yaml",
	})
	require.NoError(t, err)

	f, err = os.Create(cloudsPublicYamlPath)
	require.NoError(t, err)

	_, err = f.Write([]byte(out))
	require.NoError(t, err)

	out, err = cloudsSecureTemplate.Execute(pongo2.Context{
		"os_password": "SecuredPa$$w0rd1@",
	})
	require.NoError(t, err)

	f, err = os.Create(cloudsSecureYamlPath)
	require.NoError(t, err)

	_, err = f.Write([]byte(out))
	require.NoError(t, err)

	cl, err := GetCloudFromYAML(&ClientOpts{})
	require.NoError(t, err)
	require.Contains(t, cl.AuthInfo.AuthURL, "http://url-from-clouds-public.yaml")
	require.Contains(t, cl.AuthInfo.Password, "SecuredPa$$w0rd1")

	//err = os.Remove(cloudsYamlPath)
	//require.NoError(t, err)
	//err = os.Remove(cloudsPublicYamlPath)
	//require.NoError(t, err)
	//err = os.Remove(cloudsSecureYamlPath)
	//require.NoError(t, err)
}
