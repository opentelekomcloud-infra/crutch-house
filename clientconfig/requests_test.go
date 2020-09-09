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
