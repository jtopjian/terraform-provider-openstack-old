package openstack

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
)

type Config struct {
	IdentityEndpoint string
	UserID           string
	Username         string
	Password         string
	TenantID         string
	TenantName       string
	DomainID         string
	DomainName       string

	osClient *gophercloud.ProviderClient
}

func (c *Config) NewClient() error {

	opts := gophercloud.AuthOptions{
		IdentityEndpoint: c.IdentityEndpoint,
		UserID:           c.UserID,
		Username:         c.Username,
		Password:         c.Password,
		TenantID:         c.TenantID,
		TenantName:       c.TenantName,
		DomainID:         c.DomainID,
		DomainName:       c.DomainName,
	}

	client, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return err
	}

	c.osClient = client

	log.Printf("[INFO] Openstack Client configured for user %s", c.Username)

	return nil
}

func getClient(clientType string, d *schema.ResourceData, meta interface{}) (*gophercloud.ServiceClient, error) {
	config := meta.(*Config)
	api_version := d.Get("api_version").(string)
	region := d.Get("region").(string)

	switch clientType {
	case "block":
		return config.blockStorageClient(region, api_version)
	case "compute":
		return config.computeClient(region, api_version)
	case "network":
		return config.networkingClient(region, api_version)
	case "object":
		return config.objectStorageClient(region, api_version)
	}

	return nil, fmt.Errorf("Invalid client request: %v", clientType)
}

func (c *Config) blockStorageClient(region, api_version string) (*gophercloud.ServiceClient, error) {
	if api_version == "1" {
		return openstack.NewBlockStorageV1(c.osClient, gophercloud.EndpointOpts{
			Region: region,
		})
	}
	return nil, fmt.Errorf("api version not supported: %v", api_version)
}

func (c *Config) computeClient(region, api_version string) (*gophercloud.ServiceClient, error) {
	if api_version == "2" {
		return openstack.NewComputeV2(c.osClient, gophercloud.EndpointOpts{
			Region: region,
		})
	}
	return nil, fmt.Errorf("api version not supported: %v", api_version)
}

func (c *Config) networkingClient(region, api_version string) (*gophercloud.ServiceClient, error) {
	if api_version == "2" {
		return openstack.NewNetworkV2(c.osClient, gophercloud.EndpointOpts{
			Region: region,
		})
	}
	return nil, fmt.Errorf("api version not supported: %v", api_version)
}

func (c *Config) objectStorageClient(region, api_version string) (*gophercloud.ServiceClient, error) {
	if api_version == "1" {
		return openstack.NewObjectStorageV1(c.osClient, gophercloud.EndpointOpts{
			Region: region,
		})
	}
	return nil, fmt.Errorf("api version not supported: %v", api_version)
}
