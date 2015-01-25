package openstack

import (
	"log"

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
	//Region           string

	//ProviderClient   gophercloud.ProviderClient
}

// Client() returns a new client for accessing openstack.
//
func (c *Config) NewClient() (*gophercloud.ProviderClient, error) {

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

	provider, err := openstack.AuthenticatedClient(opts)

	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Openstack Client configured for user %s", c.Username)

	//c.ProviderClient = provider
	//return nil

	return provider, nil
}

// Currently not used
//func (p *Config) getClient() (gophercloud.ServiceClient, error) {
//	return openstack.NewComputeV2(p.ProviderClient, gophercloud.EndpointOpts{
//		Region: p.Region,
//	})
//}

// this needs updated
//func (p *Config) getNetworkApi() (network.NetworkProvider, error) {
//
//	//access := p.AccessProvider.(*gophercloud.Access)
//	access := p.ProviderClient.(*gophercloud.Access)
//
//	return network.NetworksApi(access, gophercloud.ApiCriteria{
//		Name:      "neutron",
//		UrlChoice: gophercloud.PublicURL,
//	})
//}
