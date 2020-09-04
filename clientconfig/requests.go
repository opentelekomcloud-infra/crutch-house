/*
   Ported from github.com/gophercloud/utils
*/

/*
   Original license:

   Copyright 2017 Rackspace, Inc

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package clientconfig

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"

	huaweisdk "github.com/huaweicloud/golangsdk"
	huaweicloud "github.com/huaweicloud/golangsdk/openstack"
	"github.com/huaweicloud/golangsdk/openstack/identity/v3/domains"
	"github.com/huaweicloud/golangsdk/openstack/identity/v3/endpoints"
	"github.com/huaweicloud/golangsdk/openstack/identity/v3/projects"
	"github.com/huaweicloud/golangsdk/openstack/identity/v3/services"
	"github.com/huaweicloud/golangsdk/openstack/identity/v3/tokens"
	"github.com/huaweicloud/golangsdk/pagination"
	"gopkg.in/yaml.v2"
)

// ClientOpts represents options to customize the way a client is
// configured.
type ClientOpts struct {
	// Cloud is the cloud entry in clouds.yaml to use.
	Cloud string

	// EnvPrefix allows a custom environment variable prefix to be used.
	EnvPrefix string

	// AuthInfo defines the authentication information needed to
	// authenticate to a cloud when clouds.yaml isn't used.
	AuthInfo *AuthInfo

	// RegionName is the region to create a Service Client in.
	// This will override a region in clouds.yaml or can be used
	// when authenticating directly with AuthInfo.
	RegionName string

	// EndpointType specifies whether to use the public, internal, or
	// admin endpoint of a service.
	EndpointType string

	// HTTPClient provides the ability customize the ProviderClient's
	// internal HTTP client.
	HTTPClient *http.Client

	// YAMLOpts provides the ability to pass a customized set
	// of options and methods for loading the YAML file.
	// It takes a YAMLOptsBuilder interface that is defined
	// in this file. This is optional and the default behavior
	// is to call the local LoadCloudsYAML functions defined
	// in this file.
	YAMLOpts YAMLOptsBuilder
}

// YAMLOptsBuilder defines an interface for customization when
// loading a clouds.yaml file.
type YAMLOptsBuilder interface {
	LoadCloudsYAML() (map[string]Cloud, error)
	LoadSecureCloudsYAML() (map[string]Cloud, error)
	LoadPublicCloudsYAML() (map[string]Cloud, error)
}

const getCloudFailedMessage = "could not find cloud %s"

// YAMLOpts represents options and methods to load a clouds.yaml file.
type YAMLOpts struct {
	// By default, no options are specified.
}

// LoadCloudsYAML defines how to load a clouds.yaml file.
// By default, this calls the local LoadCloudsYAML function.
func (opts YAMLOpts) LoadCloudsYAML() (map[string]Cloud, error) {
	return LoadCloudsYAML()
}

// LoadSecureCloudsYAML defines how to load a secure.yaml file.
// By default, this calls the local LoadSecureCloudsYAML function.
func (opts YAMLOpts) LoadSecureCloudsYAML() (map[string]Cloud, error) {
	return LoadSecureCloudsYAML()
}

// LoadPublicCloudsYAML defines how to load a public-secure.yaml file.
// By default, this calls the local LoadPublicCloudsYAML function.
func (opts YAMLOpts) LoadPublicCloudsYAML() (map[string]Cloud, error) {
	return LoadPublicCloudsYAML()
}

// LoadCloudsYAML will load a clouds.yaml file and return the full config.
// This is called by the YAMLOpts method. Calling this function directly
// is supported for now but has only been retained for backwards
// compatibility from before YAMLOpts was defined. This may be removed in
// the future.
func LoadCloudsYAML() (map[string]Cloud, error) {
	_, content, err := FindAndReadCloudsYAML()
	if err != nil {
		return nil, err
	}

	var clouds Clouds
	err = yaml.Unmarshal(content, &clouds)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %v", err)
	}

	return clouds.Clouds, nil
}

// LoadSecureCloudsYAML will load a secure.yaml file and return the full config.
// This is called by the YAMLOpts method. Calling this function directly
// is supported for now but has only been retained for backwards
// compatibility from before YAMLOpts was defined. This may be removed in
// the future.
func LoadSecureCloudsYAML() (map[string]Cloud, error) {
	var secureClouds Clouds

	_, content, err := FindAndReadSecureCloudsYAML()
	if err != nil {
		if err.Error() == "no secure.yaml file found" {
			// secure.yaml is optional so just ignore read error
			return secureClouds.Clouds, nil
		}
		return nil, err
	}

	err = yaml.Unmarshal(content, &secureClouds)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %v", err)
	}

	return secureClouds.Clouds, nil
}

// LoadPublicCloudsYAML will load a public-clouds.yaml file and return the full config.
// This is called by the YAMLOpts method. Calling this function directly
// is supported for now but has only been retained for backwards
// compatibility from before YAMLOpts was defined. This may be removed in
// the future.
func LoadPublicCloudsYAML() (map[string]Cloud, error) {
	var publicClouds PublicClouds

	_, content, err := FindAndReadPublicCloudsYAML()
	if err != nil {
		if err.Error() == "no clouds-public.yaml file found" {
			// clouds-public.yaml is optional so just ignore read error
			return publicClouds.Clouds, nil
		}

		return nil, err
	}

	err = yaml.Unmarshal(content, &publicClouds)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %v", err)
	}

	return publicClouds.Clouds, nil
}

// GetCloudFromYAML will return a cloud entry from a clouds.yaml file.
func GetCloudFromYAML(opts *ClientOpts) (*Cloud, error) {
	if opts.YAMLOpts == nil {
		opts.YAMLOpts = new(YAMLOpts)
	}

	yamlOpts := opts.YAMLOpts

	clouds, err := yamlOpts.LoadCloudsYAML()
	if err != nil {
		return nil, fmt.Errorf("unable to load clouds.yaml: %s", err)
	}

	// Determine which cloud to use.
	// First see if a cloud name was explicitly set in opts.
	var cloudName string
	if opts != nil && opts.Cloud != "" {
		cloudName = opts.Cloud
	}

	// Next see if a cloud name was specified as an environment variable.
	// This is supposed to override an explicit opts setting.
	envPrefix := "OS_"
	if opts != nil && opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if v := os.Getenv(envPrefix + "CLOUD"); v != "" {
		cloudName = v
	}

	var cloud *Cloud
	if cloudName != "" {
		v, ok := clouds[cloudName]
		if !ok {
			return nil, fmt.Errorf("cloud %s does not exist in clouds.yaml", cloudName)
		}
		cloud = &v
	}

	// If a cloud was not specified, and clouds only contains
	// a single entry, use that entry.
	if cloudName == "" && len(clouds) == 1 {
		for _, v := range clouds {
			cloud = &v
		}
	}

	if cloud != nil {
		// A profile points to a public cloud entry.
		// If one was specified, load a list of public clouds
		// and then merge the information with the current cloud data.
		profileName := defaultIfEmpty(cloud.Profile, cloud.Cloud)

		if profileName != "" {
			publicClouds, err := yamlOpts.LoadPublicCloudsYAML()
			if err != nil {
				return nil, fmt.Errorf("unable to load clouds-public.yaml: %s", err)
			}
			publicCloud, ok := publicClouds[profileName]
			if ok {
				cloud, err = mergeClouds(cloud, publicCloud)
				if err != nil {
					return nil, fmt.Errorf("could not merge information from clouds.yaml and clouds-public.yaml for cloud %s", profileName)
				}
			} else {
				log.Printf("cloud %s does not exist in clouds-public.yaml\n", profileName)
			}
		}
	}

	// Next, load a secure clouds file and see if a cloud entry
	// can be found or merged.
	secureClouds, err := yamlOpts.LoadSecureCloudsYAML()
	if err != nil {
		return nil, fmt.Errorf("unable to load secure.yaml: %s", err)
	}

	if secureClouds != nil {
		// If no entry was found in clouds.yaml, no cloud name was specified,
		// and only one secureCloud entry exists, use that as the cloud entry.
		if cloud == nil && cloudName == "" && len(secureClouds) == 1 {
			for _, v := range secureClouds {
				cloud = &v
			}
		}

		// Otherwise, see if the provided cloud name exists in the secure yaml file.
		secureCloud, ok := secureClouds[cloudName]
		if !ok && cloud == nil {
			// cloud == nil serves two purposes here:
			// if no entry in clouds.yaml was found and
			// if a single-entry secureCloud wasn't used.
			// At this point, no entry could be determined at all.
			return nil, fmt.Errorf(getCloudFailedMessage, cloudName)
		}

		// If secureCloud has content and it differs from the cloud entry,
		// merge the two together.
		var emptyCloud = Cloud{
			Cloud:              "",
			Profile:            "",
			AuthInfo:           nil,
			RegionName:         "",
			Regions:            nil,
			EndpointType:       "",
			Interface:          "",
			IdentityAPIVersion: "",
			VolumeAPIVersion:   "",
			Verify:             nil,
			CACertFile:         "",
			ClientCertFile:     "",
			ClientKeyFile:      "",
		}
		if !reflect.DeepEqual(emptyCloud, secureCloud) && !reflect.DeepEqual(cloud, secureCloud) {
			cloud, err = mergeClouds(secureCloud, cloud)
			if err != nil {
				return nil, fmt.Errorf("unable to merge information from clouds.yaml and secure.yaml")
			}
		}
	}

	// As an extra precaution, do one final check to see if cloud is nil.
	// We shouldn't reach this point, though.
	if cloud == nil {
		return nil, fmt.Errorf(getCloudFailedMessage, cloudName)
	}

	// Default is to verify SSL API requests
	if cloud.Verify == nil {
		iTrue := true
		cloud.Verify = &iTrue
	}

	// TODO: this is where reading vendor files should go be considered when not found in
	// clouds-public.yml
	// https://github.com/openstack/openstacksdk/tree/master/openstack/config/vendors

	// Both Interface and EndpointType are valid settings in clouds.yaml,
	// but we want to standardize on EndpointType for simplicity.
	//
	// If only Interface was set, we copy that to EndpointType to use as the setting.
	// But in all other cases, EndpointType is used and Interface is cleared.
	if cloud.Interface != "" && cloud.EndpointType == "" {
		cloud.EndpointType = cloud.Interface
	}

	cloud.Interface = ""

	return cloud, nil
}

// AuthOptions creates a gophercloud.AuthOptions structure with the
// settings found in a specific cloud entry of a clouds.yaml file or
// based on authentication settings given in ClientOpts.
//
// This attempts to be a single point of entry for all OpenStack authentication.
//
// See http://docs.openstack.org/developer/os-client-config and
// https://github.com/openstack/os-client-config/blob/master/os_client_config/config.py.
func AuthOptions(opts *ClientOpts) (huaweisdk.AuthOptionsProvider, error) {
	cloud := new(Cloud)

	// If no opts were passed in, create an empty ClientOpts.
	if opts == nil {
		opts = new(ClientOpts)
	}

	// Determine if a clouds.yaml entry should be retrieved.
	// Start by figuring out the cloud name.
	// First check if one was explicitly specified in opts.
	var cloudName string
	if opts.Cloud != "" {
		cloudName = opts.Cloud
	}

	// Next see if a cloud name was specified as an environment variable.
	envPrefix := "OS_"
	if opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if v := os.Getenv(envPrefix + "CLOUD"); v != "" {
		cloudName = v
	}

	// If a cloud name was determined, try to look it up in clouds.yaml.
	if cloudName != "" {
		// Get the requested cloud.
		var err error
		cloud, err = GetCloudFromYAML(opts)
		if err != nil {
			return nil, err
		}
	}

	// If cloud.AuthInfo is nil, then no cloud was specified.
	if cloud.AuthInfo == nil {
		// If opts.Auth is not nil, then try using the auth settings from it.
		if opts.AuthInfo != nil {
			cloud.AuthInfo = opts.AuthInfo
		}

		// If cloud.AuthInfo is still nil, then set it to an empty Auth struct
		// and rely on environment variables to do the authentication.
		if cloud.AuthInfo == nil {
			cloud.AuthInfo = new(AuthInfo)
		}
	}

	if cloud.RegionName == "" {
		cloud.RegionName = opts.RegionName
	}
	if cloud.EndpointType == "" {
		cloud.EndpointType = opts.EndpointType
	}

	cloud = setDomainIfNeeded(cloud)

	ao, err := v3auth(cloud, opts)
	if err != nil {
		return nil, err
	}
	return ao, nil
}

// v3auth creates a v3-compatible gophercloud.AuthOptions struct.
func v3auth(cloud *Cloud, opts *ClientOpts) (huaweisdk.AuthOptionsProvider, error) {
	// Environment variable overrides.
	envPrefix := "OS_"
	if opts != nil && opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if cloud.AuthInfo.AuthURL == "" {
		if v := os.Getenv(envPrefix + "AUTH_URL"); v != "" {
			cloud.AuthInfo.AuthURL = v
		}
	}

	if cloud.AuthInfo.Token == "" {
		if v := os.Getenv(envPrefix + "TOKEN"); v != "" {
			cloud.AuthInfo.Token = v
		}

		if v := os.Getenv(envPrefix + "AUTH_TOKEN"); v != "" {
			cloud.AuthInfo.Token = v
		}
	}

	if cloud.AuthInfo.Username == "" {
		if v := os.Getenv(envPrefix + "USERNAME"); v != "" {
			cloud.AuthInfo.Username = v
		}
	}

	if cloud.AuthInfo.UserID == "" {
		if v := os.Getenv(envPrefix + "USER_ID"); v != "" {
			cloud.AuthInfo.UserID = v
		}
	}

	if cloud.AuthInfo.Password == "" {
		if v := os.Getenv(envPrefix + "PASSWORD"); v != "" {
			cloud.AuthInfo.Password = v
		}
	}

	if cloud.AuthInfo.ProjectID == "" {
		if v := os.Getenv(envPrefix + "TENANT_ID"); v != "" {
			cloud.AuthInfo.ProjectID = v
		}

		if v := os.Getenv(envPrefix + "PROJECT_ID"); v != "" {
			cloud.AuthInfo.ProjectID = v
		}
	}

	if cloud.AuthInfo.ProjectName == "" {
		if v := os.Getenv(envPrefix + "TENANT_NAME"); v != "" {
			cloud.AuthInfo.ProjectName = v
		}

		if v := os.Getenv(envPrefix + "PROJECT_NAME"); v != "" {
			cloud.AuthInfo.ProjectName = v
		}
	}

	if cloud.AuthInfo.DomainID == "" {
		if v := os.Getenv(envPrefix + "DOMAIN_ID"); v != "" {
			cloud.AuthInfo.DomainID = v
		}
	}

	if cloud.AuthInfo.DomainName == "" {
		if v := os.Getenv(envPrefix + "DOMAIN_NAME"); v != "" {
			cloud.AuthInfo.DomainName = v
		}
	}

	if cloud.AuthInfo.DefaultDomain == "" {
		if v := os.Getenv(envPrefix + "DEFAULT_DOMAIN"); v != "" {
			cloud.AuthInfo.DefaultDomain = v
		}
	}

	if cloud.AuthInfo.ProjectDomainID == "" {
		if v := os.Getenv(envPrefix + "PROJECT_DOMAIN_ID"); v != "" {
			cloud.AuthInfo.ProjectDomainID = v
		}
	}

	if cloud.AuthInfo.ProjectDomainName == "" {
		if v := os.Getenv(envPrefix + "PROJECT_DOMAIN_NAME"); v != "" {
			cloud.AuthInfo.ProjectDomainName = v
		}
	}

	if cloud.AuthInfo.UserDomainID == "" {
		if v := os.Getenv(envPrefix + "USER_DOMAIN_ID"); v != "" {
			cloud.AuthInfo.UserDomainID = v
		}
	}

	if cloud.AuthInfo.UserDomainName == "" {
		if v := os.Getenv(envPrefix + "USER_DOMAIN_NAME"); v != "" {
			cloud.AuthInfo.UserDomainName = v
		}
	}

	if cloud.AuthInfo.ApplicationCredentialID == "" {
		if v := os.Getenv(envPrefix + "APPLICATION_CREDENTIAL_ID"); v != "" {
			cloud.AuthInfo.ApplicationCredentialID = v
		}
	}

	if cloud.AuthInfo.ApplicationCredentialName == "" {
		if v := os.Getenv(envPrefix + "APPLICATION_CREDENTIAL_NAME"); v != "" {
			cloud.AuthInfo.ApplicationCredentialName = v
		}
	}

	if cloud.AuthInfo.ApplicationCredentialSecret == "" {
		if v := os.Getenv(envPrefix + "APPLICATION_CREDENTIAL_SECRET"); v != "" {
			cloud.AuthInfo.ApplicationCredentialSecret = v
		}
	}

	if cloud.AuthInfo.AccessKey == "" {
		if v := os.Getenv("S3_ACCESS_KEY_ID"); v != "" {
			cloud.AuthInfo.AccessKey = v
		}
	}
	if cloud.AuthInfo.SecretKey == "" {
		if v := os.Getenv("S3_SECRET_ACCESS_KEY"); v != "" {
			cloud.AuthInfo.SecretKey = v
		}
	}

	// Check for absolute minimum requirements.
	if cloud.AuthInfo.AuthURL == "" {
		err := huaweisdk.ErrMissingInput{Argument: "auth_url"}
		return nil, err
	}

	if cloud.AuthInfo.AccessKey != "" {
		return huaweisdk.AKSKAuthOptions{
			IdentityEndpoint: cloud.AuthInfo.AuthURL,
			Region:           cloud.RegionName,
			DomainID:         cloud.AuthInfo.UserDomainID,
			ProjectId:        cloud.AuthInfo.ProjectID,
			ProjectName:      cloud.AuthInfo.ProjectName,
			AccessKey:        cloud.AuthInfo.AccessKey,
			SecretKey:        cloud.AuthInfo.SecretKey,
		}, nil
	}

	return huaweisdk.AuthOptions{
		IdentityEndpoint: cloud.AuthInfo.AuthURL,
		TokenID:          cloud.AuthInfo.Token,
		Username:         cloud.AuthInfo.Username,
		UserID:           cloud.AuthInfo.UserID,
		Password:         cloud.AuthInfo.Password,
		TenantID:         cloud.AuthInfo.ProjectID,
		TenantName:       cloud.AuthInfo.ProjectName,
		DomainID:         cloud.AuthInfo.UserDomainID,
		DomainName:       cloud.AuthInfo.UserDomainName,
	}, nil

}

func getDomainID(name string, client *huaweisdk.ServiceClient) (string, error) {
	old := client.Endpoint
	defer func() { client.Endpoint = old }()

	client.Endpoint = old + "auth/"

	opts := domains.ListOpts{
		Name: name,
	}
	allPages, err := domains.List(client, &opts).AllPages()
	if err != nil {
		return "", fmt.Errorf("list domains failed, err=%s", err)
	}

	all, err := domains.ExtractDomains(allPages)
	if err != nil {
		return "", fmt.Errorf("extract domains failed, err=%s", err)
	}

	count := len(all)
	switch count {
	case 0:
		err := &huaweisdk.ErrResourceNotFound{}
		err.ResourceType = "iam"
		err.Name = name
		return "", err
	case 1:
		return all[0].ID, nil
	default:
		err := &huaweisdk.ErrMultipleResourcesFound{}
		err.ResourceType = "iam"
		err.Name = name
		err.Count = count
		return "", err
	}
}

func getProjectID(client *huaweisdk.ServiceClient, name string) (string, error) {
	opts := projects.ListOpts{
		Name: name,
	}
	allPages, err := projects.List(client, opts).AllPages()
	if err != nil {
		return "", err
	}

	proj, err := projects.ExtractProjects(allPages)

	if err != nil {
		return "", err
	}

	if len(proj) < 1 {
		return "", fmt.Errorf("[DEBUG] cannot find the tenant: %s", name)
	}

	return proj[0].ID, nil
}

func getEntryByServiceId(entries []tokens.CatalogEntry, serviceId string) *tokens.CatalogEntry {
	if entries == nil {
		return nil
	}

	for idx := range entries {
		if entries[idx].ID == serviceId {
			return &entries[idx]
		}
	}

	return nil
}

func v3AKSKAuth(client *huaweisdk.ProviderClient, endpoint string, options huaweisdk.AKSKAuthOptions, eo huaweisdk.EndpointOpts) error {
	v3Client, err := huaweicloud.NewIdentityV3(client, eo)
	if err != nil {
		return err
	}

	if endpoint != "" {
		v3Client.Endpoint = endpoint
	}

	defer func() {
		v3Client.AKSKAuthOptions.ProjectId = options.ProjectId
		v3Client.AKSKAuthOptions.DomainID = options.DomainID
	}()
	v3Client.AKSKAuthOptions = options
	v3Client.AKSKAuthOptions.ProjectId = ""
	v3Client.AKSKAuthOptions.DomainID = ""

	if options.ProjectId == "" && options.ProjectName != "" {
		id, err := getProjectID(v3Client, options.ProjectName)
		if err != nil {
			return err
		}
		options.ProjectId = id
	}

	if options.DomainID == "" && options.Domain != "" {
		id, err := getDomainID(options.Domain, v3Client)
		if err != nil {
			options.DomainID = ""
		} else {
			options.DomainID = id
		}
	}

	client.ProjectID = options.ProjectId
	client.DomainID = options.BssDomainID
	v3Client.ProjectID = options.ProjectId

	var entries = make([]tokens.CatalogEntry, 0, 1)
	err = services.List(v3Client, services.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		serviceLst, err := services.ExtractServices(page)
		if err != nil {
			return false, err
		}

		for _, svc := range serviceLst {
			entry := tokens.CatalogEntry{
				Type: svc.Type,
				ID:   svc.ID,
			}
			entries = append(entries, entry)
		}

		return true, nil
	})

	if err != nil {
		return err
	}

	err = endpoints.List(v3Client, endpoints.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		eps, err := endpoints.ExtractEndpoints(page)
		if err != nil {
			return false, err
		}

		for _, endpoint := range eps {
			entry := getEntryByServiceId(entries, endpoint.ServiceID)

			if entry != nil {
				entry.Endpoints = append(entry.Endpoints, tokens.Endpoint{
					URL:       strings.Replace(endpoint.URL, "$(tenant_id)s", options.ProjectId, -1),
					Region:    endpoint.Region,
					Interface: string(endpoint.Availability),
					ID:        endpoint.ID,
				})
			}
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	client.EndpointLocator = func(opts huaweisdk.EndpointOpts) (string, error) {
		return huaweicloud.V3EndpointURL(&tokens.ServiceCatalog{
			Entries: entries,
		}, opts)
	}
	return nil
}

// AuthenticatedClient is a convenience function to get a new provider client
// based on a clouds.yaml entry.
func AuthenticatedClient(opts *ClientOpts) (client *huaweisdk.ProviderClient, err error) {

	ao, err := AuthOptions(opts)
	if err != nil {
		return nil, err
	}

	var authUrl string

	tokenOpts, tokenAuth := ao.(huaweisdk.AuthOptions)
	akskOpts, akskAuth := ao.(huaweisdk.AKSKAuthOptions)

	if tokenAuth {
		authUrl = tokenOpts.IdentityEndpoint
	} else if akskAuth {
		authUrl = akskOpts.IdentityEndpoint
	}
	client, err = huaweicloud.NewClient(authUrl)
	if err != nil {
		return nil, err
	}

	if akskAuth {
		err = v3AKSKAuth(client, "", akskOpts, huaweisdk.EndpointOpts{})
	} else if tokenAuth {
		err = huaweicloud.AuthenticateV3(client, &tokenOpts, huaweisdk.EndpointOpts{})
	}

	if err != nil {
		return nil, err
	}
	return
}

// NewServiceClient is a convenience function to get a new service client.
func NewServiceClient(service string, opts *ClientOpts) (*huaweisdk.ServiceClient, error) {
	cloud := new(Cloud)

	// If no opts were passed in, create an empty ClientOpts.
	if opts == nil {
		opts = new(ClientOpts)
	}

	// Determine if a clouds.yaml entry should be retrieved.
	// Start by figuring out the cloud name.
	// First check if one was explicitly specified in opts.
	var cloudName string
	if opts.Cloud != "" {
		cloudName = opts.Cloud
	}

	// Next see if a cloud name was specified as an environment variable.
	envPrefix := "OS_"
	if opts.EnvPrefix != "" {
		envPrefix = opts.EnvPrefix
	}

	if v := os.Getenv(envPrefix + "CLOUD"); v != "" {
		cloudName = v
	}

	// If a cloud name was determined, try to look it up in clouds.yaml.
	if cloudName != "" {
		// Get the requested cloud.
		var err error
		cloud, err = GetCloudFromYAML(opts)
		if err != nil {
			return nil, err
		}
	}

	// Get a Provider Client
	pClient, err := AuthenticatedClient(opts)
	if err != nil {
		return nil, err
	}

	// If an HTTPClient was specified, use it.
	if opts.HTTPClient != nil {
		pClient.HTTPClient = *opts.HTTPClient
	}

	// Determine the region to use.
	// First, check if the REGION_NAME environment variable is set.
	var region string
	if v := os.Getenv(envPrefix + "REGION_NAME"); v != "" {
		region = v
	}

	// Next, check if the cloud entry sets a region.
	if v := cloud.RegionName; v != "" {
		region = v
	}

	// Finally, see if one was specified in the ClientOpts.
	// If so, this takes precedence.
	if v := opts.RegionName; v != "" {
		region = v
	}

	// Determine the endpoint type to use.
	// First, check if the OS_INTERFACE environment variable is set.
	var endpointType string
	if v := os.Getenv(envPrefix + "INTERFACE"); v != "" {
		endpointType = v
	}

	// Next, check if the cloud entry sets an endpoint type.
	if v := cloud.EndpointType; v != "" {
		endpointType = v
	}

	// Finally, see if one was specified in the ClientOpts.
	// If so, this takes precedence.
	if v := opts.EndpointType; v != "" {
		endpointType = v
	}

	eo := huaweisdk.EndpointOpts{
		Region:       region,
		Availability: huaweisdk.Availability(GetEndpointType(endpointType)),
	}

	switch service {
	case "compute":
		return huaweicloud.NewComputeV2(pClient, eo)
	case "database":
		return huaweicloud.NewDBV1(pClient, eo)
	case "dns":
		return huaweicloud.NewDNSV2(pClient, eo)
	case "identity":
		return huaweicloud.NewIdentityV3(pClient, eo)
	case "image":
		return huaweicloud.NewImageServiceV2(pClient, eo)
	case "load-balancer":
		return huaweicloud.NewLoadBalancerV2(pClient, eo)
	case "vpc":
		return huaweicloud.NewNetworkV1(pClient, eo)
	case "network":
		return huaweicloud.NewNetworkV2(pClient, eo)
	case "object-store":
		return huaweicloud.NewObjectStorageV1(pClient, eo)
	case "cce":
		return huaweicloud.NewCCE(pClient, eo)
	case "orchestration":
		return huaweicloud.NewOrchestrationV1(pClient, eo)
	case "sharev2":
		return huaweicloud.NewSharedFileSystemV2(pClient, eo)
	case "volume":
		volumeVersion := "2"
		if v := cloud.VolumeAPIVersion; v != "" {
			volumeVersion = v
		}

		switch volumeVersion {
		case "v1", "1":
			return huaweicloud.NewBlockStorageV1(pClient, eo)
		case "v2", "2":
			return huaweicloud.NewBlockStorageV2(pClient, eo)
		case "v3", "3":
			return huaweicloud.NewBlockStorageV3(pClient, eo)
		default:
			return nil, fmt.Errorf("invalid volume API version")
		}
	}

	return nil, fmt.Errorf("unable to create a service client for %s", service)
}

// setDomainIfNeeded will set a DomainID and DomainName
// to ProjectDomain* and UserDomain* if not already set.
func setDomainIfNeeded(cloud *Cloud) *Cloud {
	if cloud.AuthInfo.DomainID != "" {
		if cloud.AuthInfo.UserDomainID == "" {
			cloud.AuthInfo.UserDomainID = cloud.AuthInfo.DomainID
		}
		if cloud.AuthInfo.ProjectDomainID == "" {
			cloud.AuthInfo.ProjectDomainID = cloud.AuthInfo.DomainID
		}
	}

	if cloud.AuthInfo.DomainName != "" {
		if cloud.AuthInfo.UserDomainName == "" {
			cloud.AuthInfo.UserDomainName = cloud.AuthInfo.DomainName
		}
		if cloud.AuthInfo.ProjectDomainName == "" {
			cloud.AuthInfo.ProjectDomainName = cloud.AuthInfo.DomainName
		}
	}
	// If Domain fields are still not set, and if DefaultDomain has a value,
	// set UserDomainID and ProjectDomainID to DefaultDomain.
	// https://github.com/openstack/osc-lib/blob/86129e6f88289ef14bfaa3f7c9cdfbea8d9fc944/osc_lib/cli/client_config.py#L117-L146
	if cloud.AuthInfo.DefaultDomain != "" {
		if cloud.AuthInfo.UserDomainName == "" && cloud.AuthInfo.UserDomainID == "" {
			cloud.AuthInfo.UserDomainID = cloud.AuthInfo.DefaultDomain
		}

		if cloud.AuthInfo.ProjectDomainName == "" && cloud.AuthInfo.ProjectDomainID == "" {
			cloud.AuthInfo.ProjectDomainID = cloud.AuthInfo.DefaultDomain
		}
	}

	return cloud
}
