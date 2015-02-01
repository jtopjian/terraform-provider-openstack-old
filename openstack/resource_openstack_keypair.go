package openstack

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/keypairs"
)

func resourceKeypair() *schema.Resource {
	return &schema.Resource{
		Create: resourceKeypairCreate,
		Read:   resourceKeypairRead,
		Update: nil,
		Delete: resourceKeypairDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"public_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			// Region is defined per-instance due to how gophercloud
			// handles the region -- not until a provider is returned.
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			// read-only / exported
			"fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"user_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceKeypairCreate(d *schema.ResourceData, meta interface{}) error {
	client, err := getComputeClient(d, meta)
	if err != nil {
		return err
	}

	keyName := d.Get("name").(string)
	publicKey := d.Get("public_key").(string)

	opts := &keypairs.CreateOpts{
		Name:      keyName,
		PublicKey: publicKey,
	}

	newKey, err := keypairs.Create(client, opts).Extract()
	if err != nil {
		return err
	}

	d.SetId(newKey.Name)
	if err := setKeypairDetails(client, newKey.Name, d); err != nil {
		return err
	}

	return nil

}

func resourceKeypairRead(d *schema.ResourceData, meta interface{}) error {
	client, err := getComputeClient(d, meta)
	if err != nil {
		return err
	}

	if err := setKeypairDetails(client, d.Id(), d); err != nil {
		return err
	}

	return nil
}

func resourceKeypairDelete(d *schema.ResourceData, meta interface{}) error {
	client, err := getComputeClient(d, meta)
	if err != nil {
		return err
	}

	if err := keypairs.Delete(client, d.Id()).ExtractErr(); err != nil {
		return err
	}

	return nil
}

func setKeypairDetails(client *gophercloud.ServiceClient, keyName string, d *schema.ResourceData) error {
	key, err := keypairs.Get(client, keyName).Extract()
	if err != nil {
		return err
	}
	log.Printf("[INFO] Key info: %v", key)

	d.Set("name", key.Name)
	d.Set("public_key", key.PublicKey)
	d.Set("fingerprint", key.Fingerprint)
	d.Set("user_id", key.UserID)

	return nil
}
