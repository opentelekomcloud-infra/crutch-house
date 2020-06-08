package testing

import (
	"fmt"
	"net/http"
	"testing"

	fake "github.com/huaweicloud/golangsdk/openstack/networking/v1/common"
	th "github.com/huaweicloud/golangsdk/testhelper"
	secgroupsv1 "github.com/opentelekomcloud-infra/crutch-house/openstack/networking/v1/secgroups"
)

func TestListSecurityGroup(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	th.Mux.HandleFunc("/v1/85636478b0bd8e67e89469c7749d4127/security-groups", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, `
{
    "security_groups": [
        {
            "id": "16b6e77a-08fa-42c7-aa8b-106c048884e6", 
            "name": "qq", 
            "description": "qq", 
            "vpc_id": "3ec3b33f-ac1c-4630-ad1c-7dba1ed79d85", 
           
            "security_group_rules": [
                {
                    "direction": "egress", 
                    "ethertype": "IPv4", 
                    "id": "369e6499-b2cb-4126-972a-97e589692c62", 
                    "description": "",
                    "security_group_id": "16b6e77a-08fa-42c7-aa8b-106c048884e6"
                }, 
                {
                    "direction": "ingress", 
                    "ethertype": "IPv4", 
                    "id": "0222556c-6556-40ad-8aac-9fd5d3c06171", 
                    "description": "",
                    "remote_group_id": "16b6e77a-08fa-42c7-aa8b-106c048884e6", 
                    "security_group_id": "16b6e77a-08fa-42c7-aa8b-106c048884e6"
                }
            ]
        }, 
        {
            "id": "9c0f56be-a9ac-438c-8c57-fce62de19419", 
            "name": "default", 
            "description": "qq", 
            "vpc_id": "13551d6b-755d-4757-b956-536f674975c0", 
           
            "security_group_rules": [
                {
                    "direction": "egress", 
                    "ethertype": "IPv4", 
                    "id": "95479e0a-e312-4844-b53d-a5e4541b783f", 
                    "description": "",
                    "security_group_id": "9c0f56be-a9ac-438c-8c57-fce62de19419"
                }, 
                {
                    "direction": "ingress", 
                    "ethertype": "IPv4", 
                    "id": "0c4a2336-b036-4fa2-bc3c-1a291ed4c431",
                    "description": "", 
                    "remote_group_id": "9c0f56be-a9ac-438c-8c57-fce62de19419", 
                    "security_group_id": "9c0f56be-a9ac-438c-8c57-fce62de19419"
                }
            ]
        }
    ]
}
			`)
	})

	actual, err := secgroupsv1.List(fake.ServiceClient(), secgroupsv1.ListOpts{})
	if err != nil {
		t.Errorf("Failed to extract security groups: %v", err)
	}

	expected := []secgroupsv1.SecurityGroup{
		{
			ID:          "16b6e77a-08fa-42c7-aa8b-106c048884e6",
			Name:        "qq",
			Description: "qq",
			Rules: []secgroupsv1.Rule{
				{
					ID:          "369e6499-b2cb-4126-972a-97e589692c62",
					Description: "",
					SecGroupID:  "16b6e77a-08fa-42c7-aa8b-106c048884e6",
					Direction:   secgroupsv1.Egress,
					Ethertype:   secgroupsv1.IPv4,
				},
				{
					ID:             "0222556c-6556-40ad-8aac-9fd5d3c06171",
					Description:    "",
					SecGroupID:     "16b6e77a-08fa-42c7-aa8b-106c048884e6",
					Direction:      secgroupsv1.Ingress,
					Ethertype:      secgroupsv1.IPv4,
					RemoteIPPrefix: "",
					RemoteGroupID:  "16b6e77a-08fa-42c7-aa8b-106c048884e6",
				},
			},
		},
		{
			ID:          "9c0f56be-a9ac-438c-8c57-fce62de19419",
			Name:        "default",
			Description: "qq",
			Rules: []secgroupsv1.Rule{
				{
					ID:          "95479e0a-e312-4844-b53d-a5e4541b783f",
					Description: "",
					SecGroupID:  "9c0f56be-a9ac-438c-8c57-fce62de19419",
					Direction:   secgroupsv1.Egress,
					Ethertype:   secgroupsv1.IPv4,
				},
				{
					ID:            "0c4a2336-b036-4fa2-bc3c-1a291ed4c431",
					Description:   "",
					SecGroupID:    "9c0f56be-a9ac-438c-8c57-fce62de19419",
					Direction:     secgroupsv1.Ingress,
					Ethertype:     secgroupsv1.IPv4,
					RemoteGroupID: "9c0f56be-a9ac-438c-8c57-fce62de19419",
				},
			},
		},
	}

	th.AssertDeepEquals(t, expected, actual)
}
