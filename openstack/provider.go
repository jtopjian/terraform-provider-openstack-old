package openstack

import (
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"auth_url": &schema.Schema{
				Type:        schema.TypeString,
				DefaultFunc: envDefaultFunc("OS_AUTH_URL"),
				Required:    true,
			},

			"username": &schema.Schema{
				Type:        schema.TypeString,
				DefaultFunc: envDefaultFunc("OS_USERNAME"),
				Required:    true,
			},

			"password": &schema.Schema{
				Type:        schema.TypeString,
				DefaultFunc: envDefaultFunc("OS_PASSWORD"),
				Required:    true,
			},

			"tenant_id": &schema.Schema{
				Type:        schema.TypeString,
				DefaultFunc: envDefaultFunc("OS_TENANT_ID"),
				Required:    true,
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"openstack_network":         resourceNetwork(),
			"openstack_subnet":          resourceSubnet(),
			"openstack_router":          resourceRouter(),
			"openstack_security_group":  resourceSecurityGroup(),
			"openstack_compute":         resourceCompute(),
			"openstack_lbaas":           resourceLBaaS(),
			"openstack_firewall":        resourceFirewall(),
			"openstack_firewall_policy": resourceFirewallPolicy(),
			"openstack_firewall_rule":   resourceFirewallRule(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func envDefaultFunc(k string) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		if v := os.Getenv(k); v != "" {
			return v, nil
		}

		return nil, nil
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Auth:     d.Get("auth_url").(string),
		User:     d.Get("username").(string),
		Password: d.Get("password").(string),
		TenantId: d.Get("tenant_id").(string),
	}

	if err := config.NewClient(); err != nil {
		return nil, err
	}

	return &config, nil
}
