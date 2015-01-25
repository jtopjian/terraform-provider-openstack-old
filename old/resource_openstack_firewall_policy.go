package openstack

import (
	"log"

	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
)

func resourceFirewallPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceFirewallPolicyCreate,
		Read:   resourceFirewallPolicyRead,
		Delete: resourceFirewallPolicyDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"audited": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"shared": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"rules": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
		},
	}
}

func resourceFirewallPolicyCreate(d *schema.ResourceData, meta interface{}) error {

	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	access := p.AccessProvider.(*gophercloud.Access)

	v := d.Get("rules").(*schema.Set)
	rules := make([]string, v.Len())
	for i, v := range v.List() {
		rules[i] = v.(string)
	}

	policyConfiguration := network.NewFirewallPolicy{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Audited:     d.Get("audited").(bool),
		Shared:      d.Get("shared").(bool),
		TenantId:    access.Token.Tenant.Id,
		Rules:       rules,
	}

	log.Printf("[DEBUG] Create firewall policy: %#v", policyConfiguration)

	firewall, err := networksApi.CreateFirewallPolicy(policyConfiguration)
	if err != nil {
		return err
	}

	d.SetId(firewall.Id)

	return nil
}

func resourceFirewallPolicyRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Retrieve information about firewall policy: %s", d.Id())

	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	n, err := networksApi.GetFirewallPolicy(d.Id())
	if err != nil {
		httpError, ok := err.(*perigee.UnexpectedResponseCodeError)
		if !ok {
			return err
		}

		if httpError.Actual == 404 {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", n.Name)

	return nil
}

func resourceFirewallPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Destroy firewall policy: %s", d.Id())

	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	return networksApi.DeleteFirewallPolicy(d.Id())
}
