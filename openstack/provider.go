package openstack

import (
	"errors"
	"fmt"
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/compute/v2/flavors"
	"github.com/rackspace/gophercloud/openstack/compute/v2/images"
	"github.com/rackspace/gophercloud/pagination"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"identity_endpoint": &schema.Schema{
				Type:        schema.TypeString,
				DefaultFunc: envDefaultFunc("OS_AUTH_URL"),
				Required:    true,
			},

			"user_id": &schema.Schema{
				Type:        schema.TypeString,
				DefaultFunc: envDefaultFunc("OS_USERID"),
				Optional:    true,
			},

			"username": &schema.Schema{
				Type:        schema.TypeString,
				DefaultFunc: envDefaultFunc("OS_USERNAME"),
				Optional:    true,
			},

			"password": &schema.Schema{
				Type:        schema.TypeString,
				DefaultFunc: envDefaultFunc("OS_PASSWORD"),
				Required:    true,
			},

			"tenant_id": &schema.Schema{
				Type:        schema.TypeString,
				DefaultFunc: envDefaultFunc("OS_TENANT_ID"),
				Optional:    true,
			},

			"tenant_name": &schema.Schema{
				Type:        schema.TypeString,
				DefaultFunc: envDefaultFunc("OS_TENANT_NAME"),
				Optional:    true,
			},

			"domain_id": &schema.Schema{
				Type:        schema.TypeString,
				DefaultFunc: envDefaultFunc("OS_DOMAIN_ID"),
				Optional:    true,
			},

			"domain_name": &schema.Schema{
				Type:        schema.TypeString,
				DefaultFunc: envDefaultFunc("OS_DOMAIN_NAME"),
				Optional:    true,
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"openstack_instance":    resourceInstance(),
			"openstack_keypair":     resourceKeypair(),
			"openstack_floating_ip": resourceFloatingIP(),
			"openstack_secgroup":    resourceSecgroup(),
			"openstack_volume":      resourceVolume(),
			//"openstack_network":         resourceNetwork(),
			//"openstack_subnet":          resourceSubnet(),
			//"openstack_router":          resourceRouter(),
			//"openstack_security_group":  resourceSecurityGroup(),
			//"openstack_lbaas":           resourceLBaaS(),
			//"openstack_firewall":        resourceFirewall(),
			//"openstack_firewall_policy": resourceFirewallPolicy(),
			//"openstack_firewall_rule":   resourceFirewallRule(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func envDefaultFunc(k string) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		if v := os.Getenv(k); v != "" {
			return v, nil
		}

		return "", nil
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {

	user_id := d.Get("user_id").(string)
	username := d.Get("username").(string)
	if user_id == "" && username == "" {
		return nil, errors.New("At least one of user_id or username must be specified")
	}

	tenant_id := d.Get("tenant_id").(string)
	tenant_name := d.Get("tenant_name").(string)
	if tenant_id == "" && tenant_name == "" {
		return nil, errors.New("At least one of tenant_id or tenant_name must be specified")
	}

	config := Config{
		IdentityEndpoint: d.Get("identity_endpoint").(string),
		UserID:           d.Get("user_id").(string),
		Username:         d.Get("username").(string),
		Password:         d.Get("password").(string),
		TenantID:         d.Get("tenant_id").(string),
		TenantName:       d.Get("tenant_name").(string),
		DomainID:         d.Get("domain_id").(string),
		DomainName:       d.Get("domain_name").(string),
	}

	return config.NewClient()
}

func getComputeClient(d *schema.ResourceData, meta interface{}) (*gophercloud.ServiceClient, error) {
	provider := meta.(*gophercloud.ProviderClient)
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})

	if err != nil {
		return nil, err
	}

	return client, nil
}

func getBlockStorageClient(d *schema.ResourceData, meta interface{}) (*gophercloud.ServiceClient, error) {
	provider := meta.(*gophercloud.ProviderClient)
	client, err := openstack.NewBlockStorageV1(provider, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})

	if err != nil {
		return nil, err
	}

	return client, nil
}

// helper functions

// images
func getImageID(client *gophercloud.ServiceClient, d *schema.ResourceData) (string, error) {
	imageID := d.Get("image_id").(string)
	imageName := d.Get("image_name").(string)
	if imageID == "" {
		pager := images.ListDetail(client, nil)

		pager.EachPage(func(page pagination.Page) (bool, error) {
			imageList, err := images.ExtractImages(page)

			if err != nil {
				return false, err
			}

			for _, i := range imageList {
				if i.Name == imageName {
					imageID = i.ID
				}
			}
			return true, nil
		})

		if imageID == "" {
			return "", fmt.Errorf("Unable to find image: %v", imageName)
		}
	}

	if imageID == "" && imageName == "" {
		return "", fmt.Errorf("Neither an image ID nor an image name were able to be determined.")
	}

	return imageID, nil
}

func getImage(client *gophercloud.ServiceClient, imageId string) (*images.Image, error) {
	return images.Get(client, imageId).Extract()
}

// flavors
func getFlavorID(client *gophercloud.ServiceClient, d *schema.ResourceData) (string, error) {
	flavorID := d.Get("flavor_id").(string)
	flavorName := d.Get("flavor_name").(string)
	if flavorID == "" {
		pager := flavors.ListDetail(client, nil)

		pager.EachPage(func(page pagination.Page) (bool, error) {
			flavorList, err := flavors.ExtractFlavors(page)

			if err != nil {
				return false, err
			}

			for _, f := range flavorList {
				if f.Name == flavorName {
					flavorID = f.ID
				}
			}
			return true, nil
		})

		if flavorID == "" {
			return "", fmt.Errorf("Unable to find flavor: %v", flavorName)
		}
	}

	if flavorID == "" && flavorName == "" {
		return "", fmt.Errorf("Neither a flavor ID nor a flavor name were able to be determined.")
	}

	return flavorID, nil
}

func getFlavor(client *gophercloud.ServiceClient, flavorId string) (*flavors.Flavor, error) {
	return flavors.Get(client, flavorId).Extract()
}
