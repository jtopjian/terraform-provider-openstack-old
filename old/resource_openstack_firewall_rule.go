package openstack

import (
	"log"

	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
)

func resourceFirewallRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceFirewallRuleCreate,
		Read:   resourceFirewallRuleRead,
		Delete: resourceFirewallRuleDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"action": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"ip_version": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  4,
			},
			"source_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"destination_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"source_port": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"destination_port": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"shared": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func resourceFirewallRuleCreate(d *schema.ResourceData, meta interface{}) error {

	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	access := p.AccessProvider.(*gophercloud.Access)

	ruleConfiguration := network.NewFirewallRule{
		Name:                 d.Get("name").(string),
		Description:          d.Get("description").(string),
		Protocol:             d.Get("protocol").(string),
		Action:               d.Get("action").(string),
		IpVersion:            d.Get("ip_version").(int),
		SourceIpAddress:      d.Get("source_ip_address").(string),
		DestinationIpAddress: d.Get("destination_ip_address").(string),
		SourcePort:           d.Get("source_port").(string),
		DestinationPort:      d.Get("destination_port").(string),
		Shared:               d.Get("shared").(bool),
		Enabled:              d.Get("enabled").(bool),
		TenantId:             access.Token.Tenant.Id,
	}

	log.Printf("[DEBUG] Create firewall rule: %#v", ruleConfiguration)

	firewall, err := networksApi.CreateFirewallRule(ruleConfiguration)
	if err != nil {
		return err
	}

	d.SetId(firewall.Id)

	return nil
}

func resourceFirewallRuleRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Retrieve information about firewall rule: %s", d.Id())

	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	n, err := networksApi.GetFirewallRule(d.Id())
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

func resourceFirewallRuleDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Destroy firewall rule: %s", d.Id())

	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	return networksApi.DeleteFirewallRule(d.Id())
}
