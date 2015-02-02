package openstack

import (
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceFloatingIP() *schema.Resource {
	return &schema.Resource{
		Create: resourceFloatingIPCreate,
		Read:   resourceFloatingIPRead,
		Update: resourceFloatingIPUpdate,
		Delete: resourceFloatingIPDelete,

		Schema: map[string]*schema.Schema{
			"pool": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"instance_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			// Region is defined per-instance due to how gophercloud
			// handles the region -- not until a provider is returned.
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			// network_service specifies which network provider to use
			// Either "nova-network" or "neutron"
			"network_service": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "neutron",
			},

			// read-only / exported
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"fixed_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceFloatingIPCreate(d *schema.ResourceData, meta interface{}) error {
	networkService := d.Get("network_service")
	if networkService == "nova-network" {
		client, err := getComputeClient(d, meta)
		if err != nil {
			return err
		}

		pool := d.Get("pool").(string)

		newFip, err := createNovaNetworkFloatingIP(client, pool)
		if err != nil {
			return err
		}

		d.SetId(strconv.Itoa(newFip.Id))
		if err := setNovaNetworkFloatingIPDetails(client, newFip.Id, d); err != nil {
			return err
		}

		// if instance_id is specified, associate it with an instance
		if newFip.InstanceId != "" {
			associateNovaNetworkFloatingIP(client, newFip.InstanceId, newFip)
		}
	}

	return nil
}

func resourceFloatingIPRead(d *schema.ResourceData, meta interface{}) error {
	networkService := d.Get("network_service")
	if networkService == "nova-network" {
		client, err := getComputeClient(d, meta)
		if err != nil {
			return err
		}

		fId, err := strconv.Atoi(d.Id())
		if err != nil {
			return err
		}

		if err := setNovaNetworkFloatingIPDetails(client, fId, d); err != nil {
			return err
		}
	}

	return nil
}

func resourceFloatingIPUpdate(d *schema.ResourceData, meta interface{}) error {
	networkService := d.Get("network_service")
	if networkService == "nova-network" {
		client, err := getComputeClient(d, meta)
		if err != nil {
			return err
		}

		fId, err := strconv.Atoi(d.Id())
		if err != nil {
			return err
		}

		fip, err := getNovaNetworkFloatingIP(client, fId)

		// the only thing that can be updated is the instance id
		// So check to see if it's associated with an instance
		// disassociate it
		// then associate it with the new instance id
		if d.HasChange("instance_id") {
			oldId, newId := d.GetChange("instance_id")
			log.Printf("[INFO] Old ID: %v", oldId)
			log.Printf("[INFO] New ID: %v", newId)
			if oldId.(string) != "" && oldId.(string) != "0" {
				log.Printf("[INFO] Attempting to disassociate")
				if err := disassociateNovaNetworkFloatingIP(client, oldId.(string), fip); err != nil {
					return err
				}
			}
			if newId.(string) != "" && newId.(string) != "0" {
				log.Printf("[INFO] Attempting to associate")
				if err := associateNovaNetworkFloatingIP(client, newId.(string), fip); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func resourceFloatingIPDelete(d *schema.ResourceData, meta interface{}) error {
	networkService := d.Get("network_service")
	if networkService == "nova-network" {
		client, err := getComputeClient(d, meta)
		if err != nil {
			return err
		}

		fId, err := strconv.Atoi(d.Id())
		if err != nil {
			return err
		}

		if err := deleteNovaNetworkFloatingIP(client, fId); err != nil {
			return err
		}
	}

	return nil
}
