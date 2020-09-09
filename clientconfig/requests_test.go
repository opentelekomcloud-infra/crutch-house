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

//func TestGetCloudFromPublic(t *testing.T) {
//	cloudsPublicYamlPath := fmt.Sprintf("resources/%s.yaml", utils.RandomString(15, "clouds-public"))
//
//	_ = os.Setenv("OS_CLIENT_VENDOR_FILE", cloudsPublicYamlPath)
//	_ = os.Setenv("OS_CLOUD", "test")
//
//	tpl, err := pongo2.FromFile("resources/clouds-public.yaml.j2")
//	require.NoError(t, err)
//
//	out, err := tpl.Execute(pongo2.Context{
//		"os_auth_url":     "http://test-me.com/v5",
//		"os_project_name": "eu-de",
//		"os_username":     "otc",
//		"os_password":     "Qwerty123!",
//		"os_domain_name":  "OTC987414257102518"})
//	require.NoError(t, err)
//
//	f, err := os.Create(cloudsPublicYamlPath)
//	require.NoError(t, err)
//	defer f.Close()
//
//	_, err = f.Write([]byte(out))
//	require.NoError(t, err)
//
//	cl, err := GetCloudFromYAML(&ClientOpts{})
//	require.NoError(t, err)
//	require.Contains(t, cl.AuthInfo.Password, "Qwerty123!")
//}
