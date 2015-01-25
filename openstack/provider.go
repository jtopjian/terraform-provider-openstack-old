package openstack

import (
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"identity_endpoint": &schema.Schema{
				Type:        schema.TypeString,
				DefaultFunc: envDefaultFunc("OS_AUTH_URL"),
				Required:    true,
			},

			// One of these two is required
			// TODO: add logic to support that
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

			// One of these two is required
			// TODO: add logic to support that
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
			//"openstack_network":         resourceNetwork(),
			//"openstack_subnet":          resourceSubnet(),
			//"openstack_router":          resourceRouter(),
			//"openstack_security_group":  resourceSecurityGroup(),
			"openstack_compute": resourceCompute(),
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
