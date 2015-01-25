package openstack

import (
	"fmt"
	"log"

	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
)

func resourceRouter() *schema.Resource {
	return &schema.Resource{
		Create: resourceRouterCreate,
		Read:   resourceRouterRead,
		Update: resourceRouterUpdate,
		Delete: resourceRouterDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"external_gateway": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
				ForceNew: true,
			},
			"external_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"subnets": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true, // TODO handle update of the subnets array
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
		},
	}
}

func resourceRouterCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	newRouter := network.NewRouter{
		Name:         d.Get("name").(string),
		AdminStateUp: true,
	}

	if d.Get("external_gateway").(bool) {
		// Try to find external network
		if len(d.Get("external_id").(string)) == 0 {
			networks, err := networksApi.GetNetworks()
			if err != nil {
				return err
			}

			var externalId string
			for _, element := range networks {
				if element.IsExternal {
					externalId = element.Id
				}
			}
			if externalId == "" {
				return fmt.Errorf("Cannot find external network while creating an external gateway")
			}

			d.Set("external_id", externalId)
		}

		externalGateway := network.ExternalGateway{
			NetworkId: d.Get("external_id").(string),
		}

		newRouter.ExternalGateway = externalGateway
	}

	createdRouter, err := networksApi.CreateRouter(newRouter)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] New router created: %#v", createdRouter)

	d.SetId(createdRouter.Id)

	// TODO wait for router status

	// Attach subnets
	if v := d.Get("subnets").(*schema.Set); v.Len() > 0 {
		subnets := make([]string, v.Len())
		for i, v := range v.List() {
			subnets[i] = v.(string)
		}

		for _, subnet := range subnets {
			_, err := networksApi.AddRouterInterface(d.Id(), subnet)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func resourceRouterDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	ports, err := networksApi.GetPorts()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Destroy router: %s", d.Id())

	for _, port := range ports {
		if port.DeviceId == d.Id() {
			err = networksApi.RemoveRouterInterface(d.Id(), port.PortId)
			if err != nil {
				return err
			}
		}
	}

	return networksApi.DeleteRouter(d.Id())
}

func resourceRouterRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	router, err := networksApi.GetRouter(d.Id())
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

	d.Set("name", router.Name)
	d.Set("external_id", router.ExternalGateway.NetworkId)

	return nil
}

func resourceRouterUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Updating router: %#v", d.Id())

	if d.HasChange("name") {
		_, err := networksApi.UpdateRouter(network.Router{
			Id:   d.Id(),
			Name: d.Get("name").(string),
		})

		if err != nil {
			return err
		}
	}

	return nil
}
