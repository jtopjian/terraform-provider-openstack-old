package openstack

import (
	"log"

	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
)

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkCreate,
		Read:   resourceNetworkRead,
		Update: resourceNetworkUpdate,
		Delete: resourceNetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceNetworkCreate(d *schema.ResourceData, meta interface{}) error {

	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	access := p.AccessProvider.(*gophercloud.Access)

	networkConfiguration := network.NewNetwork{
		Name:         d.Get("name").(string),
		AdminStateUp: true,
		TenantId:     access.Token.Tenant.Id,
	}

	log.Printf("[DEBUG] Create network: %#v", networkConfiguration)

	newNetwork, err := networksApi.CreateNetwork(networkConfiguration)
	if err != nil {
		return err
	}

	d.SetId(newNetwork.Id)

	return nil
}

func resourceNetworkDelete(d *schema.ResourceData, meta interface{}) error {

	log.Printf("[DEBUG] Destroy network: %s", d.Id())

	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	return networksApi.DeleteNetwork(d.Id())
}

func resourceNetworkRead(d *schema.ResourceData, meta interface{}) error {

	log.Printf("[DEBUG] Retrieve information about network: %s", d.Id())

	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	n, err := networksApi.GetNetwork(d.Id())
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

func resourceNetworkUpdate(d *schema.ResourceData, meta interface{}) error {

	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	d.Partial(true)

	if d.HasChange("name") {
		_, err := networksApi.UpdateNetwork(d.Id(), network.UpdatedNetwork{
			Name: d.Get("name").(string),
		})

		if err != nil {
			return err
		}

		d.SetPartial("name")
	}

	d.Partial(false)

	return nil
}
