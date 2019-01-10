package keystone

import (
	"gopkg.in/niedbalski/goose.v3/client"
	"gopkg.in/niedbalski/goose.v3/errors"
	goosehttp "gopkg.in/niedbalski/goose.v3/http"
)

// API URL parts.
const (
	apiDomains	= "domains"
	ApiGroups	= "groups"
	ApiRegions	= "regions"
	ApiProjects	= "projects"
	ApiUsers	= "users"
)

// Client provides a means to access the OpenStack Compute Service.
type Client struct {
	client client.Client
}

// New creates a new Client.
func New(client client.Client) *Client {
	return &Client{client}
}

type Domain struct {
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	ID          string `json:"id"`
	Links       struct {
		Self string `json:"self"`
	} `json:"links"`
	Name string `json:"name"`
}

type DomainList struct {
	Domains []Domain `json:"domains"`
	Links struct {
		Next     interface{} `json:"next"`
		Previous interface{} `json:"previous"`
		Self     string      `json:"self"`
	} `json:"links"`
}

type Region struct {
	Description string `json:"description"`
	ID          string `json:"id"`
	Links       struct {
		Self string `json:"self"`
	} `json:"links"`
	ParentRegionID interface{} `json:"parent_region_id"`
}

type RegionList struct {
	Regions []Region `json:"regions"`
	Links struct {
		Next     interface{} `json:"next"`
		Previous interface{} `json:"previous"`
		Self     string      `json:"self"`
	} `json:"links"`
}

type User struct {
	DomainID string `json:"domain_id"`
	Enabled  bool   `json:"enabled"`
	ID       string `json:"id"`
	Links    struct {
		Self string `json:"self"`
	} `json:"links"`
	Name              string      `json:"name"`
	PasswordExpiresAt interface{} `json:"password_expires_at"`
}

type UserList struct {
	Links struct {
		Next     interface{} `json:"next"`
		Previous interface{} `json:"previous"`
		Self     string      `json:"self"`
	} `json:"links"`
	Users []User `json:"users"`
}

type Project struct {
	IsDomain    bool        `json:"is_domain"`
	Description interface{} `json:"description"`
	DomainID    string      `json:"domain_id"`
	Enabled     bool        `json:"enabled"`
	ID          string      `json:"id"`
	Links       struct {
		Self string `json:"self"`
	} `json:"links"`
	Name     string        `json:"name"`
	ParentID interface{}   `json:"parent_id"`
	Tags     []interface{} `json:"tags"`
}

type ProjectList struct {
	Links struct {
		Next     interface{} `json:"next"`
		Previous interface{} `json:"previous"`
		Self     string      `json:"self"`
	} `json:"links"`

	Projects []Project `json:"projects"`
}

type Group struct {
	Description string `json:"description"`
	DomainID    string `json:"domain_id"`
	ID          string `json:"id"`
	Links       struct {
		Self string `json:"self"`
	} `json:"links"`
	Name string `json:"name"`
}

type GroupList struct {
	Links struct {
		Self     string      `json:"self"`
		Previous interface{} `json:"previous"`
		Next     interface{} `json:"next"`
	} `json:"links"`
	Groups []Group `json:"groups"`
}

func (c *Client) ListDomains() ([]Domain, error) {
	var response DomainList
	requestData := goosehttp.RequestData{RespValue: &response}
	err := c.client.SendRequest(client.GET, "identity", "v3", apiDomains, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to list domains")
	}
	return response.Domains, nil
}

func (c *Client) ListUsers() ([]User, error) {
	var response UserList
	requestData := goosehttp.RequestData{RespValue: &response}
	err := c.client.SendRequest(client.GET, "identity", "v3", ApiUsers, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to list users")
	}
	return response.Users, nil
}

func (c *Client) ListProjects() ([]Project, error) {
	var response ProjectList
	requestData := goosehttp.RequestData{RespValue: &response}
	err := c.client.SendRequest(client.GET, "identity", "v3", ApiProjects, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to list projects")
	}
	return response.Projects, nil
}

func (c *Client) ListGroups() ([]Group, error) {
	var response GroupList
	requestData := goosehttp.RequestData{RespValue: &response}
	err := c.client.SendRequest(client.GET, "identity", "v3", ApiGroups, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to list groups")
	}
	return response.Groups, nil
}

func (c *Client) ListRegions() ([]Region, error) {
	var response RegionList
	requestData := goosehttp.RequestData{RespValue: &response}
	err := c.client.SendRequest(client.GET, "identity", "v3", ApiRegions, &requestData)
	if err != nil {
		return nil, errors.Newf(err, "failed to list regions")
	}
	return response.Regions, nil
}