package openstack

import (
	"log"
	"os"
	"strings"

	"github.com/haklop/gophercloud-extensions/network"
	"github.com/rackspace/gophercloud"
)

type Config struct {
	Auth       string `mapstructure:"auth_url"`
	User       string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	ApiKey     string `mapstructure:"api_key"`
	TenantId   string `mapstructure:"tenant_id"`
	TenantName string `mapstructure:"tenant_name"`

	AccessProvider gophercloud.AccessProvider
}

// Client() returns a new client for accessing openstack.
//
func (c *Config) NewClient() error {

	if v := os.Getenv("OS_AUTH_URL"); v != "" {
		c.Auth = v
	}
	if v := os.Getenv("OS_USERNAME"); v != "" {
		c.User = v
	}
	if v := os.Getenv("OS_PASSWORD"); v != "" {
		c.Password = v
	}
	if v := os.Getenv("OS_TENANT_ID"); v != "" {
		c.TenantId = v
	}
	if v := os.Getenv("OS_TENANT_NAME"); v != "" {
		c.TenantName = v
	}

	// OpenStack's auto-generated openrc.sh files do not append the suffix
	// /tokens to the authentication URL. This ensures it is present when
	// specifying the URL.
	if strings.Contains(c.Auth, "://") && !strings.HasSuffix(c.Auth, "/tokens") {
		c.Auth += "/tokens"
	}

	accessProvider, err := gophercloud.Authenticate(
		c.Auth,
		gophercloud.AuthOptions{
			ApiKey:     c.ApiKey,
			Username:   c.User,
			Password:   c.Password,
			TenantName: c.TenantName,
			TenantId:   c.TenantId,
		},
	)

	c.AccessProvider = accessProvider

	if err != nil {
		return err
	}

	log.Printf("[INFO] Openstack Client configured for user %s", c.User)

	return nil
}

func (p *Config) getServersApi() (gophercloud.CloudServersProvider, error) {
	return gophercloud.ServersApi(p.AccessProvider, gophercloud.ApiCriteria{
		Name:      "nova",
		UrlChoice: gophercloud.PublicURL,
	})
}

func (p *Config) getNetworkApi() (network.NetworkProvider, error) {

	access := p.AccessProvider.(*gophercloud.Access)

	return network.NetworksApi(access, gophercloud.ApiCriteria{
		Name:      "neutron",
		UrlChoice: gophercloud.PublicURL,
	})
}
