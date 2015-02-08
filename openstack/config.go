package openstack

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
)

type Config struct {
	IdentityEndpoint        string
	UserID                  string
	Username                string
	Password                string
	TenantID                string
	TenantName              string
	DomainID                string
	DomainName              string
	BlockStorageAPIVersion  string
	ComputeAPIVersion       string
	NetworkingAPIVersion    string
	ObjectStorageAPIVersion string

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
	region := d.Get("region").(string)

	switch clientType {
	case "block":
		return config.blockStorageClient(region)
	case "compute":
		return config.computeClient(region)
	case "network":
		return config.networkingClient(region)
	case "object":
		return config.objectStorageClient(region)
	}

	return nil, fmt.Errorf("Invalid client request: %v", clientType)
}

func (c *Config) blockStorageClient(region string) (*gophercloud.ServiceClient, error) {
	if c.BlockStorageAPIVersion == "1" {
		return openstack.NewBlockStorageV1(c.osClient, gophercloud.EndpointOpts{
			Region: region,
		})
	}
	return nil, fmt.Errorf("block storage api version not supported: %v", c.BlockStorageAPIVersion)
}

func (c *Config) computeClient(region string) (*gophercloud.ServiceClient, error) {
	if c.ComputeAPIVersion == "2" {
		return openstack.NewComputeV2(c.osClient, gophercloud.EndpointOpts{
			Region: region,
		})
	}
	return nil, fmt.Errorf("compute api version not supported: %v", c.ComputeAPIVersion)
}

func (c *Config) networkingClient(region string) (*gophercloud.ServiceClient, error) {
	if c.NetworkingAPIVersion == "2" {
		return openstack.NewNetworkV2(c.osClient, gophercloud.EndpointOpts{
			Region: region,
		})
	}
	return nil, fmt.Errorf("network api version not supported: %v", c.NetworkingAPIVersion)
}

func (c *Config) objectStorageClient(region string) (*gophercloud.ServiceClient, error) {
	if c.ObjectStorageAPIVersion == "1" {
		return openstack.NewObjectStorageV1(c.osClient, gophercloud.EndpointOpts{
			Region: region,
		})
	}
	return nil, fmt.Errorf("object api version not supported: %v", c.ObjectStorageAPIVersion)
}
