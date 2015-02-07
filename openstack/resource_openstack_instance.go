package openstack

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
)

func resourceInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceInstanceCreate,
		Read:   resourceInstanceRead,
		Update: resourceInstanceUpdate,
		Delete: resourceInstanceDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"image_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"image_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"flavor_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"flavor_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
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

			"network": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true, // TODO handle update
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uuid": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"port": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"fixed_ip": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
				Set: resourceInstanceNetworkHash,
			},

			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
				Default:  nil,
			},

			"config_drive": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"admin_pass": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"key_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			// read-only
			"created": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"updated": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"network_info": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
		},
	}
}

func resourceInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	if err := checkParameters(d); err != nil {
		return err
	}

	client, err := getComputeClient(d, meta)
	if err != nil {
		return err
	}

	userData := []byte("")
	if v := d.Get("user_data"); v != nil {
		userData = []byte(v.(string))
	}

	imageID, err := getImageID(client, d)
	if err != nil {
		return err
	}

	flavorID, err := getFlavorID(client, d)
	if err != nil {
		return err
	}

	var createOpts servers.CreateOptsBuilder
	createOpts = &servers.CreateOpts{
		Name:           d.Get("name").(string),
		ImageRef:       imageID,
		FlavorRef:      flavorID,
		SecurityGroups: buildInstanceSecurityGroups(d),
		Networks:       buildInstanceNetworks(d),
		UserData:       userData,
		AdminPass:      d.Get("admin_pass").(string),
		ConfigDrive:    d.Get("config_drive").(bool),
		Metadata:       buildInstanceMetadata(d),
	}

	if keyName, ok := d.Get("key_name").(string); ok && keyName != "" {
		createOpts = &keypairs.CreateOptsExt{
			createOpts,
			keyName,
		}
	}

	newServer, err := servers.Create(client, createOpts).Extract()
	if err != nil {
		return err
	}

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

	// get full info about the new server
	d.SetId(newServer.ID)
	if err := setServerDetails(client, newServer.ID, d); err != nil {
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

func resourceInstanceRead(d *schema.ResourceData, meta interface{}) error {
	if err := checkParameters(d); err != nil {
		return err
	}

	client, err := getComputeClient(d, meta)
	if err != nil {
		return err
	}

	if err := setServerDetails(client, d.Id(), d); err != nil {
		return err
	}

	return nil
}

func resourceInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	if err := checkParameters(d); err != nil {
		return err
	}

	client, err := getComputeClient(d, meta)
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

		// FIXME: TEST
		if res := servers.ConfirmResize(client, server.ID); res.Err != nil {
			return res.Err
		}

		d.SetPartial("flavor_ref")
	}

	d.Partial(false)

	return nil
}

func resourceInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	if err := checkParameters(d); err != nil {
		return err
	}

	client, err := getComputeClient(d, meta)
	if err != nil {
		return err
	}

	server, _ := servers.Get(client, d.Id()).Extract()

	servers.Delete(client, server.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE", "ERROR"},
		Target:     "DELETED",
		Refresh:    waitForServerState(client, server),
		Timeout:    30 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()

	return err
}

func waitForServerState(client *gophercloud.ServiceClient, server *servers.Server) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		latest, err := servers.Get(client, server.ID).Extract()
		if err != nil {
			httpStatus := err.(*perigee.UnexpectedResponseCodeError)
			if httpStatus.Actual == 404 {
				return "", "DELETED", nil
			}
			return nil, "", err
		}

		return latest, latest.Status, nil

	}
}

func checkParameters(d *schema.ResourceData) error {
	imageID := d.Get("image_id").(string)
	imageName := d.Get("image_name").(string)
	if imageID == "" && imageName == "" {
		return errors.New("At least one of image_id or image_name is required.")
	}

	flavorID := d.Get("flavor_id").(string)
	flavorName := d.Get("flavor_name").(string)
	if flavorID == "" && flavorName == "" {
		return errors.New("At least one of flavor_id or flavor_name is required.")
	}

	return nil
}

func setServerDetails(client *gophercloud.ServiceClient, serverID string, d *schema.ResourceData) error {
	server, err := servers.Get(client, serverID).Extract()
	if err != nil {
		return err
	}
	log.Printf("[INFO] Server info: %v", server)

	flavor, err := getFlavor(client, server.Flavor["id"].(string))
	if err != nil {
		return err
	}
	log.Printf("[INFO] Flavor info: %v", flavor)

	image, err := getImage(client, server.Image["id"].(string))
	if err != nil {
		return err
	}
	log.Printf("[INFO] Image info: %v", image)

	d.Set("name", server.Name)
	d.Set("id", server.ID)
	d.Set("tenant_id", server.TenantID)
	d.Set("user_id", server.UserID)
	d.Set("updated", server.Updated)
	d.Set("created", server.Created)
	d.Set("key_name", server.KeyName)
	d.Set("image_id", image.ID)
	d.Set("image_name", image.Name)
	d.Set("flavor_id", flavor.ID)
	d.Set("flavor_name", flavor.Name)

	addrs := make(map[string]string)
	for pool, pool_info := range server.Addresses {
		mac_name := fmt.Sprintf("%s_mac", pool)
		ipv4_name := fmt.Sprintf("%s_ipv4", pool)
		ipv6_name := fmt.Sprintf("%s_ipv6", pool)
		p := pool_info.([]interface{})
		for _, v := range p {
			v2 := v.(map[string]interface{})
			addrs[mac_name] = v2["OS-EXT-IPS-MAC:mac_addr"].(string)
			if v2["version"] == 4.0 {
				addrs[ipv4_name] = v2["addr"].(string)
			} else if v2["version"] == 6.0 {
				addrs[ipv6_name] = fmt.Sprintf("[%s]", v2["addr"].(string))
			}
		}
	}

	d.Set("network_info", addrs)
	log.Printf("[INFO] addrs: %v", addrs)

	return nil
}

func resourceInstanceNetworkHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["uuid"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["port"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["fixed_ip"].(string)))
	return hashcode.String(buf.String())
}

func buildInstanceNetworks(d *schema.ResourceData) []servers.Network {
	var networks []servers.Network
	if v, ok := d.GetOk("network"); ok {
		log.Printf("[INFO] network: %v", v)
		vs := v.(*schema.Set).List()
		if len(vs) > 0 {
			for _, v := range vs {
				net := v.(map[string]interface{})
				networks = append(networks, servers.Network{
					UUID:    net["uuid"].(string),
					Port:    net["port"].(string),
					FixedIP: net["fixed_ip"].(string),
				})
			}
		}
	} else {
		nets := d.Get("networks").(*schema.Set)
		for _, v := range nets.List() {
			log.Printf("[INFO] network: %v", v)
			networks = append(networks, servers.Network{UUID: v.(string)})
		}
	}
	return networks
}

func buildInstanceSecurityGroups(d *schema.ResourceData) []string {
	secGroups := d.Get("security_groups").(*schema.Set)
	var securityGroups []string
	for _, v := range secGroups.List() {
		securityGroups = append(securityGroups, v.(string))
	}
	return securityGroups
}

func buildInstanceMetadata(d *schema.ResourceData) map[string]string {
	metadata := make(map[string]string)
	if m, ok := d.GetOk("metadata"); ok {
		if len(m.(map[string]interface{})) > 1 {
			for k, v := range m.(map[string]interface{}) {
				metadata[k] = v.(string)
			}
		} else {
			metadata = nil
		}
	}
	return metadata
}
