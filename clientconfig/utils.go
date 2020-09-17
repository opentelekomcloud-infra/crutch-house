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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/opentelekomcloud/gophertelekomcloud"
)

const DefaultEndpointType = string(golangsdk.AvailabilityPublic)

// defaultIfEmpty is a helper function to make it cleaner to set default value
// for strings.
func defaultIfEmpty(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// mergeClouds merges two Clouds recursively (the AuthInfo also gets merged).
// In case both Clouds define a value, the value in the 'override' cloud takes precedence
func mergeClouds(override, cloud interface{}) (*Cloud, error) {
	overrideJson, err := json.Marshal(override)
	if err != nil {
		return nil, err
	}
	cloudJson, err := json.Marshal(cloud)
	if err != nil {
		return nil, err
	}
	var overrideInterface interface{}
	err = json.Unmarshal(overrideJson, &overrideInterface)
	if err != nil {
		return nil, err
	}
	var cloudInterface interface{}
	err = json.Unmarshal(cloudJson, &cloudInterface)
	if err != nil {
		return nil, err
	}
	var mergedCloud Cloud
	mergedInterface := mergeInterfaces(overrideInterface, cloudInterface)
	mergedJson, err := json.Marshal(mergedInterface)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(mergedJson, &mergedCloud); err != nil {
		return nil, err
	}
	return &mergedCloud, nil
}

// merges two interfaces. In cases where a value is defined for both 'overridingInterface' and
// 'inferiorInterface' the value in 'overridingInterface' will take precedence.
func mergeInterfaces(overridingInterface, inferiorInterface interface{}) interface{} {
	switch overriding := overridingInterface.(type) {
	case map[string]interface{}:
		interfaceMap, ok := inferiorInterface.(map[string]interface{})
		if !ok {
			return overriding
		}
		for k, v := range interfaceMap {
			if overridingValue, ok := overriding[k]; ok {
				overriding[k] = mergeInterfaces(overridingValue, v)
			} else {
				overriding[k] = v
			}
		}
	case []interface{}:
		list, ok := inferiorInterface.([]interface{})
		if !ok {
			return overriding
		}
		for i := range list {
			overriding = append(overriding, list[i])
		}
		return overriding
	case nil:
		// mergeClouds(nil, map[string]interface{...}) -> map[string]interface{...}
		v, ok := inferiorInterface.(map[string]interface{})
		if ok {
			return v
		}
	}
	// We don't want to override with empty values
	if reflect.DeepEqual(overridingInterface, nil) || reflect.DeepEqual(reflect.Zero(reflect.TypeOf(overridingInterface)).Interface(), overridingInterface) {
		return inferiorInterface
	} else {
		return overridingInterface
	}
}

// FindAndReadCloudsYAML attempts to locate a clouds.yaml file in the following
// locations:
//
// 1. OS_CLIENT_CONFIG_FILE
// 2. Current directory.
// 3. unix-specific user_config_dir (~/.config/openstack/clouds.yaml)
// 4. unix-specific site_config_dir (/etc/openstack/clouds.yaml)
//
// If found, the contents of the file is returned.
func FindAndReadCloudsYAML() (string, []byte, error) {
	// OS_CLIENT_CONFIG_FILE
	if v := os.Getenv("OS_CLIENT_CONFIG_FILE"); v != "" {
		if ok := fileExists(v); ok {
			content, err := ioutil.ReadFile(v)
			return v, content, err
		}
	}

	return FindAndReadYAML("clouds.yaml")
}

func FindAndReadPublicCloudsYAML() (string, []byte, error) {
	// OS_CLIENT_VENDOR_FILE
	if v := os.Getenv("OS_CLIENT_VENDOR_FILE"); v != "" {
		if ok := fileExists(v); ok {
			content, err := ioutil.ReadFile(v)
			return v, content, err
		}
	}
	return FindAndReadYAML("clouds-public.yaml")
}

func FindAndReadSecureCloudsYAML() (string, []byte, error) {
	// OS_CLIENT_SECURE_FILE
	if v := os.Getenv("OS_CLIENT_SECURE_FILE"); v != "" {
		if ok := fileExists(v); ok {
			content, err := ioutil.ReadFile(v)
			return v, content, err
		}
	}
	return FindAndReadYAML("secure.yaml")
}

func FindAndReadYAML(yamlFile string) (string, []byte, error) {
	// current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", nil, fmt.Errorf("unable to determine working directory: %s", err)
	}

	filename := filepath.Join(cwd, yamlFile)
	if ok := fileExists(filename); ok {
		content, err := ioutil.ReadFile(filename)
		return filename, content, err
	}

	// unix user config directory: ~/.config/openstack.
	if currentUser, err := user.Current(); err == nil {
		homeDir := currentUser.HomeDir
		if homeDir != "" {
			filename := filepath.Join(homeDir, ".config/openstack/"+yamlFile)
			if ok := fileExists(filename); ok {
				content, err := ioutil.ReadFile(filename)
				return filename, content, err
			}
		}
	}

	// unix-specific site config directory: /etc/openstack.
	filename = "/etc/openstack/" + yamlFile
	if ok := fileExists(filename); ok {
		content, err := ioutil.ReadFile(filename)
		return filename, content, err
	}

	return "", nil, fmt.Errorf("no " + yamlFile + " file found")
}

// fileExists checks for the existence of a file at a given location.
func fileExists(filename string) bool {
	if _, err := os.Stat(filename); err == nil {
		return true
	}
	return false
}

var validEndpointTypes = []string{"public", "internal", "admin"}

// GetEndpointType is a helper method to determine the endpoint type
// requested by the user.
func GetEndpointType(endpointType string) string {
	for _, eType := range validEndpointTypes {
		if strings.HasPrefix(endpointType, eType) {
			return eType
		}
	}
	return DefaultEndpointType
}
