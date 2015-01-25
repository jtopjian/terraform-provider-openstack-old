package openstack

import (
	"log"

	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
)

func resourceSubnet() *schema.Resource {
	return &schema.Resource{
		Create: resourceSubnetCreate,
		Read:   resourceSubnetRead,
		Update: resourceSubnetUpdate,
		Delete: resourceSubnetDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"cidr": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"ip_version": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"enable_dhcp": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
			},
			"network_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceSubnetCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	newSubnet := network.NewSubnet{
		NetworkId:  d.Get("network_id").(string),
		Name:       d.Get("name").(string),
		Cidr:       d.Get("cidr").(string),
		EnableDhcp: d.Get("enable_dhcp").(bool),
		IPVersion:  d.Get("ip_version").(int),
	}

	log.Printf("[DEBUG] Creating subnet: %#v", newSubnet)

	createdSubnet, err := networksApi.CreateSubnet(newSubnet)
	if err != nil {
		return err
	}

	d.SetId(createdSubnet.Id)

	return nil
}

func resourceSubnetDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Destroy subnet: %s", d.Id())

	return networksApi.DeleteSubnet(d.Id())
}

func resourceSubnetRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Retrieve information about subnet: %s", d.Id())

	subnet, err := networksApi.GetSubnet(d.Id())
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

	d.Set("name", subnet.Name)
	d.Set("enable_dhcp", subnet.EnableDhcp)

	return nil
}

func resourceSubnetUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	desiredSubnet := network.UpdatedSubnet{}

	if d.HasChange("name") {
		desiredSubnet.Name = d.Get("name").(string)
	}

	if d.HasChange("enable_dhcp") {
		desiredSubnet.EnableDhcp = d.Get("enable_dhcp").(bool)
	}

	_, err = networksApi.UpdateSubnet(d.Id(), desiredSubnet)
	return err
}
