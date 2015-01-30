package openstack

import (
	"crypto/sha1"
	//"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	//"github.com/haklop/gophercloud-extensions/network"
)

func resourceCompute() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeCreate,
		Read:   resourceComputeRead,
		Update: resourceComputeUpdate,
		Delete: resourceComputeDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"image_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"image_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"flavor_ref": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"security_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true, // TODO handle update
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"user_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				// just stash the hash for state & diff comparisons
				StateFunc: func(v interface{}) string {
					switch v.(type) {
					case string:
						hash := sha1.Sum([]byte(v.(string)))
						return hex.EncodeToString(hash[:])
					default:
						return ""
					}
				},
			},

			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"networks": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true, // TODO handle update
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			// No idea how to do this yet.
			//"metadata": &schema.Schema{
			//},

			// No idea how to do this yet.
			//"personality": &schema.Schema{
			//},

			"config_drive": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"admin_pass": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			// Region is defined per-instance due to how gophercloud
			// handles the region -- not until a provider is returned.
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			// defined in gophercloud compute extensions
			"key_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true, // TODO handle update
			},

			// defined in haklop's network extensions
			// Neutron only, I think
			"floating_ip_pool": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"floating_ip": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceComputeCreate(d *schema.ResourceData, meta interface{}) error {
	// check for required parameters
	// is there a better way to do this?
	imageID := d.Get("image_id").(string)
	imageName := d.Get("image_name").(string)
	if imageID == "" && imageName == "" {
		return errors.New("At least one of image_id or image_name is required.")
	}

	client, err := getClient(d, meta)
	if err != nil {
		return err
	}

	nets := d.Get("networks").(*schema.Set)
	var networks []servers.Network
	for _, v := range nets.List() {
		networks = append(networks, servers.Network{UUID: v.(string)})
	}

	secGroups := d.Get("security_groups").(*schema.Set)
	var securityGroups []string
	for _, v := range secGroups.List() {
		securityGroups = append(securityGroups, v.(string))
	}

	// API needs it to be base64 encoded.
	/*
	   userData := ""
	   if v := d.Get("user_data"); v != nil {
	     userData = base64.StdEncoding.EncodeToString([]byte(v.(string)))
	   }
	*/

	// figure out the image
	if imageID == "" {
		// image_name has to be set due to prior checking
		// look up all accessible images and find a match
		imageID, err = GetImageIDByName(client, imageName)
		if err != nil {
			return err
		}
	}

	baseOpts := &servers.CreateOpts{
		Name:           d.Get("name").(string),
		ImageRef:       imageID,
		FlavorRef:      d.Get("flavor_ref").(string),
		SecurityGroups: securityGroups,
		Networks:       networks,
		// Need to convert this to type byte
		//UserData:       userData,
	}

	keyName := d.Get("key_name").(string)
	keyOpts := keypairs.CreateOptsExt{
		CreateOptsBuilder: baseOpts,
		KeyName:           keyName,
	}

	newServer, err := servers.Create(client, keyOpts).Extract()

	if err != nil {
		return err
	}

	d.SetId(newServer.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"BUILD"},
		Target:     "ACTIVE",
		Refresh:    waitForServerState(client, newServer),
		Timeout:    30 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()

	if err != nil {
		return err
	}

	// FIXME: add floating IP support
	/*
	   pool := d.Get("floating_ip_pool").(string)
	   if len(pool) > 0 {
	     var newIp gophercloud.FloatingIp
	     hasFloatingIps := false

	     floatingIps, err := serversApi.ListFloatingIps()
	     if err != nil {
	       return err
	     }

	     for _, element := range floatingIps {
	       // use first floating ip available on the pool
	       if element.Pool == pool && element.InstanceId == "" {
	         newIp = element
	         hasFloatingIps = true
	       }
	     }

	     // if there is no available floating ips, try to create a new one
	     if !hasFloatingIps {
	       newIp, err = serversApi.CreateFloatingIp(pool)
	       if err != nil {
	         return err
	       }
	     }

	     err = serversApi.AssociateFloatingIp(newServer.Id, newIp)
	     if err != nil {
	       return err
	     }

	     d.Set("floating_ip", newIp.Ip)

	     // Initialize the connection info
	     d.SetConnInfo(map[string]string{
	       "type": "ssh",
	       "host": newIp.Ip,
	     })
	   }
	*/

	return nil
}

func resourceComputeDelete(d *schema.ResourceData, meta interface{}) error {

	client, err := getClient(d, meta)
	if err != nil {
		return err
	}

	server, _ := servers.Get(client, d.Id()).Extract()

	servers.Delete(client, server.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE", "ERROR"},
		Target:     "",
		Refresh:    waitForServerState(client, server),
		Timeout:    30 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()

	return err
}

func resourceComputeUpdate(d *schema.ResourceData, meta interface{}) error {

	client, err := getClient(d, meta)
	if err != nil {
		return err
	}

	server, _ := servers.Get(client, d.Id()).Extract()

	d.Partial(true)

	if d.HasChange("name") {
		_, err := servers.Update(client, server.ID, servers.UpdateOpts{
			Name: d.Get("name").(string),
		}).Extract()

		if err != nil {
			return err
		}

		d.SetPartial("name")
	}

	if d.HasChange("flavor_ref") {
		opts := &servers.ResizeOpts{
			FlavorRef: d.Get("flavor_ref").(string),
		}

		if res := servers.Resize(client, server.ID, opts); res.Err != nil {
			return res.Err
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"ACTIVE", "RESIZE"},
			Target:     "VERIFY_RESIZE",
			Refresh:    waitForServerState(client, server),
			Timeout:    30 * time.Minute,
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = stateConf.WaitForState()

		if err != nil {
			return err
		}

		// FIXME: proper error checking
		if res := servers.ConfirmResize(client, server.ID); res.Err != nil {
			return res.Err
		}

		d.SetPartial("flavor_ref")
	}

	d.Partial(false)

	return nil
}

func resourceComputeRead(d *schema.ResourceData, meta interface{}) error {
	client, err := getClient(d, meta)
	if err != nil {
		return err
	}

	server, err := servers.Get(client, d.Id()).Extract()
	if err != nil {
		return nil
	}

	// TODO check networks, seucrity groups and floating ip

	d.Set("name", server.Name)
	d.Set("flavor_ref", server.Flavor["ID"])
	d.Set("image_id", server.Image["ID"])
	d.Set("image_name", server.Image["Name"])

	return nil
}

func waitForServerState(client *gophercloud.ServiceClient, server *servers.Server) resource.StateRefreshFunc {

	return func() (interface{}, string, error) {
		latest, err := servers.Get(client, server.ID).Extract()
		if err != nil {
			return nil, "", err
		}

		return latest, latest.Status, nil

	}
}

func getClient(d *schema.ResourceData, meta interface{}) (*gophercloud.ServiceClient, error) {

	provider := meta.(*gophercloud.ProviderClient)
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})

	if err != nil {
		return nil, err
	}

	return client, nil

}
