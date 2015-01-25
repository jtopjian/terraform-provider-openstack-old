package openstack

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
)

func resourceFirewall() *schema.Resource {
	return &schema.Resource{
		Create: resourceFirewallCreate,
		Read:   resourceFirewallRead,
		Delete: resourceFirewallDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"policy_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"admin_state_up": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"shared": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceFirewallCreate(d *schema.ResourceData, meta interface{}) error {

	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	access := p.AccessProvider.(*gophercloud.Access)

	firewallConfiguration := network.NewFirewall{
		Name:         d.Get("name").(string),
		Description:  d.Get("description").(string),
		PolicyId:     d.Get("policy_id").(string),
		AdminStateUp: d.Get("admin_state_up").(bool),
		Shared:       d.Get("shared").(bool),
		TenantId:     access.Token.Tenant.Id,
	}

	log.Printf("[DEBUG] Create firewall: %#v", firewallConfiguration)

	firewall, err := networksApi.CreateFirewall(firewallConfiguration)
	if err != nil {
		return err
	}

	d.SetId(firewall.Id)

	return nil
}

func resourceFirewallRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Retrieve information about firewall: %s", d.Id())

	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	n, err := networksApi.GetFirewall(d.Id())
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

func resourceFirewallDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Destroy firewall: %s", d.Id())

	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	err = networksApi.DeleteFirewall(d.Id())

	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"DELETING"},
		Target:     "DELETED",
		Refresh:    WaitForFirewallDeletion(networksApi, d.Id()),
		Timeout:    2 * time.Minute,
		Delay:      0,
		MinTimeout: 2 * time.Second,
	}

	_, err = stateConf.WaitForState()

	return err
}

func WaitForFirewallDeletion(api network.NetworkProvider, id string) resource.StateRefreshFunc {

	return func() (interface{}, string, error) {
		fw, err := api.GetFirewall(id)
		log.Printf("[DEBUG] Get firewall %s => %#v", id, fw)

		if err != nil {
			httpStatus := err.(*perigee.UnexpectedResponseCodeError)
			log.Printf("[DEBUG] Get firewall %s status is %d", id, httpStatus.Actual)

			if httpStatus.Actual == 404 {
				log.Printf("[DEBUG] Firewall %s is actually deleted", id)
				return "", "DELETED", nil
			}

			return nil, "", errors.New(fmt.Sprintf("Unexpected status code %d", httpStatus.Actual))
		}

		log.Printf("[DEBUG] Firewall %s deletion is pending", id)
		return fw, "DELETING", nil
	}
}
