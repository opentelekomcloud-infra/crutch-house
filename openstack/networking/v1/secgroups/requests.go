package secgroups

import (
	"fmt"
	"reflect"

	"github.com/huaweicloud/golangsdk"
	"github.com/huaweicloud/golangsdk/pagination"
)

type ListOptsBuilder interface {
	ToSecGroupsListQuery() (string, error)
}

type ListOpts struct {
	SecurityGroup
	Marker string `q:"marker"`
	Limit  string `q:"limit"`
	VpcID  string `q:"vpc_id"`
}

func (opts ListOpts) ToSecGroupsListQuery() (string, error) {
	queryOpts := ListOpts{VpcID: opts.VpcID}
	q, err := golangsdk.BuildQueryString(queryOpts)
	if err != nil {
		return "", err
	}
	return q.String(), err
}

func List(c *golangsdk.ServiceClient, opts ListOpts) ([]SecurityGroup, error) {
	q, err := opts.ToSecGroupsListQuery()
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s%s", rootURL(c), q)
	pages, err := pagination.NewPager(c, u, func(r pagination.PageResult) pagination.Page {
		return SecGroupPage{pagination.LinkedPageBase{PageResult: r}}
	}).AllPages()

	allVpcs, err := ExtractSecGroups(pages)
	if err != nil {
		return nil, err
	}

	return filterSecGroups(allVpcs, opts)
}

func filterSecGroups(src []SecurityGroup, opts ListOpts) ([]SecurityGroup, error) {

	var refinedSecGroups []SecurityGroup
	var matched bool
	m := map[string]interface{}{}

	if opts.Name != "" {
		m["name"] = opts.Name
	}
	if opts.Description != "" {
		m["description"] = opts.Description
	}

	if len(m) > 0 && len(src) > 0 {
		for _, grp := range src {
			matched = true

			for key, value := range m {
				if sVal := getStructField(&grp, key); !(sVal == value) {
					matched = false
				}
			}

			if matched {
				refinedSecGroups = append(refinedSecGroups, grp)
			}
		}

	} else {
		refinedSecGroups = src
	}

	return refinedSecGroups, nil
}

func getStructField(v *SecurityGroup, field string) string {
	r := reflect.ValueOf(v)
	f := reflect.Indirect(r).FieldByName(field)
	return f.String()
}
