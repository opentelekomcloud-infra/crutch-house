package secgroups

import (
	"github.com/huaweicloud/golangsdk"
	"github.com/huaweicloud/golangsdk/pagination"
)

type Direction string
type EtherType string
type Protocol string

const (
	Egress  Direction = "egress"
	Ingress Direction = "ingress"

	IPv4 EtherType = "IPv4"
	IPv6 EtherType = "IPv6"

	All  Protocol = ""
	ICMP Protocol = "icmp"
	TCP  Protocol = "tcp"
	UDP  Protocol = "udp"
)

type PortRange struct {
	Min uint16
	Max uint16
}

// Route is a possible route in a vpc.
type Rule struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	SecGroupID  string `json:"sec_group_id"`
	// Rule direction
	Direction Direction `json:"direction"`
	// IP protocol version
	Ethertype EtherType `json:"ethertype"`
	// Protocol type. If the parameter is left blank, all protocols are supported.
	Protocol Protocol `json:"protocol"`

	PortRangeMin uint16 `json:"port_range_min"`
	PortRangeMax uint16 `json:"port_range_max"`

	RemoteIPPrefix string `json:"remote_ip_prefix,omitempty"`
	RemoteGroupID  string `json:"remote_group_id,omitempty"`
}

type SecurityGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Rules       []Rule `json:"security_group_rules"`
}

// SecGroupPage is the page returned by a pager when traversing over a
// collection of security groups.
type SecGroupPage struct {
	pagination.LinkedPageBase
}

// NextPageURL is invoked when a paginated collection of vpcs has reached
// the end of a page and the pager seeks to traverse over a new one. In order
// to do this, it needs to construct the next page's URL.
func (r SecGroupPage) NextPageURL() (string, error) {
	var s struct {
		Links []golangsdk.Link `json:"vpcs_links"`
	}
	err := r.ExtractInto(&s)
	if err != nil {
		return "", err
	}
	return golangsdk.ExtractNextURL(s.Links)
}

// IsEmpty checks whether a SecGroupPage struct is empty.
func (r SecGroupPage) IsEmpty() (bool, error) {
	is, err := ExtractSecGroups(r)
	return len(is) == 0, err
}

// ExtractSecGroups accepts a Page struct, specifically a SecGroupPage struct,
// and extracts the elements into a slice of SecurityGroup structs. In other words,
// a generic collection is mapped into a relevant slice.
func ExtractSecGroups(r pagination.Page) ([]SecurityGroup, error) {
	var s struct {
		SecurityGroups []SecurityGroup `json:"sec_groups"`
	}
	err := (r.(SecGroupPage)).ExtractInto(&s)
	return s.SecurityGroups, err
}

type commonResult struct {
	golangsdk.Result
}

// Extract is a function that accepts a result and extracts a vpc.
func (r commonResult) Extract() (*SecurityGroup, error) {
	var s struct {
		SecurityGroup *SecurityGroup `json:"sec_group"`
	}
	err := r.ExtractInto(&s)
	return s.SecurityGroup, err
}

// CreateResult represents the result of a create operation. Call its Extract
// method to interpret it as a Vpc.
type CreateResult struct {
	commonResult
}

// GetResult represents the result of a get operation. Call its Extract
// method to interpret it as a Vpc.
type GetResult struct {
	commonResult
}

// UpdateResult represents the result of an update operation. Call its Extract
// method to interpret it as a Vpc.
type UpdateResult struct {
	commonResult
}

// DeleteResult represents the result of a delete operation. Call its ExtractErr
// method to determine if the request succeeded or failed.
type DeleteResult struct {
	golangsdk.ErrResult
}
