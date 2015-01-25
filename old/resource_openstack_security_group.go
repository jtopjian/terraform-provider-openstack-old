package openstack

import (
	"fmt"

	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
)

func resourceSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceSecurityGroupCreate,
		Read:   resourceSecurityGroupRead,
		Delete: resourceSecurityGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"rule": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"direction": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"remote_ip_prefix": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"port_range_min": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"port_range_max": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"rule_id": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func resourceSecurityGroupCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	access := p.AccessProvider.(*gophercloud.Access)

	newSecurityGroup, err := networksApi.CreateSecurityGroup(network.NewSecurityGroup{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		TenantId:    access.Token.Tenant.Id,
	})
	if err != nil {
		return err
	}

	d.SetId(newSecurityGroup.Id)

	rulesCount := d.Get("rule.#").(int)
	rules := make([]*network.SecurityGroupRule, 0, rulesCount)
	for i := 0; i < rulesCount; i++ {
		prefix := fmt.Sprintf("rule.%d", i)

		var rule network.SecurityGroupRule
		rule.Direction = d.Get(prefix + ".direction").(string)
		rule.RemoteIpPrefix = d.Get(prefix + ".remote_ip_prefix").(string)
		rule.PortRangeMin = d.Get(prefix + ".port_range_min").(int)
		rule.PortRangeMax = d.Get(prefix + ".port_range_max").(int)
		rule.Protocol = d.Get(prefix + ".protocol").(string)

		rules = append(rules, &rule)
	}

	for _, rule := range rules {
		rule.SecurityGroupId = d.Id()
		rule.TenantId = access.Token.Tenant.Id

		// TODO store rules id for allowed updates
		_, err := networksApi.CreateSecurityGroupRule(*rule)
		if err != nil {
			return err
		}
	}

	return nil
}

func resourceSecurityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	return networksApi.DeleteSecurityGroup(d.Id())
}

func resourceSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	_, err = networksApi.GetSecurityGroup(d.Id())
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

	// TODO check rules

	return nil
}
