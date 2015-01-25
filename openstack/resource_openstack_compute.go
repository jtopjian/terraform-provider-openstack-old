package openstack

import (
	"time"
	"crypto/sha1"
	"encoding/hex"
	"encoding/base64"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
)

func resourceCompute() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeCreate,
		Read:   resourceComputeRead,
		Update: resourceComputeUpdate,
		Delete: resourceComputeDelete,

		Schema: map[string]*schema.Schema{
			"image_ref": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"key_pair_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true, // TODO handle update
			},

			"flavor_ref": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
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

			"security_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true, // TODO handle update
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"floating_ip_pool": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"floating_ip": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"user_data": &schema.Schema{
				Type: schema.TypeString,
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
		},
	}
}

func resourceComputeCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	serversApi, err := p.getServersApi()
	if err != nil {
		return err
	}

	v := d.Get("networks").(*schema.Set)
	networks := make([]gophercloud.NetworkConfig, v.Len())
	for i, v := range v.List() {
		networks[i] = gophercloud.NetworkConfig{v.(string)}
	}

	v = d.Get("security_groups").(*schema.Set)
	securityGroup := make([]map[string]interface{}, v.Len())
	for i, v := range v.List() {
		securityGroup[i] = map[string]interface{}{"name": v.(string)}
	}

	// API needs it to be base64 encoded.
	userData := ""
	if v := d.Get("user_data"); v!= nil {
		userData = base64.StdEncoding.EncodeToString([]byte(v.(string)))
	}

	newServer, err := serversApi.CreateServer(gophercloud.NewServer{
		Name:          d.Get("name").(string),
		ImageRef:      d.Get("image_ref").(string),
		FlavorRef:     d.Get("flavor_ref").(string),
		KeyPairName:   d.Get("key_pair_name").(string),
		Networks:      networks,
		SecurityGroup: securityGroup,
		UserData:      userData,
	})

	if err != nil {
		return err
	}

	d.SetId(newServer.Id)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"BUILD"},
		Target:     "ACTIVE",
		Refresh:    WaitForServerState(serversApi, d.Id()),
		Timeout:    30 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()

	if err != nil {
		return err
	}

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

	return nil
}

func resourceComputeDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	serversApi, err := p.getServersApi()
	if err != nil {
		return err
	}

	err = serversApi.DeleteServerById(d.Id())
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE", "ERROR"},
		Target:     "DELETED",
		Refresh:    WaitForServerState(serversApi, d.Id()),
		Timeout:    30 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()

	return err
}

func resourceComputeUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	serversApi, err := p.getServersApi()
	if err != nil {
		return err
	}

	d.Partial(true)

	if d.HasChange("name") {
		_, err := serversApi.UpdateServer(d.Id(), gophercloud.NewServerSettings{
			Name: d.Get("name").(string),
		})

		if err != nil {
			return err
		}

		d.SetPartial("name")
	}

	if d.HasChange("flavor_ref") {
		err := serversApi.ResizeServer(d.Id(), d.Get("name").(string), d.Get("flavor_ref").(string), "")

		if err != nil {
			return err
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"ACTIVE", "RESIZE"},
			Target:     "VERIFY_RESIZE",
			Refresh:    WaitForServerState(serversApi, d.Id()),
			Timeout:    30 * time.Minute,
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = stateConf.WaitForState()

		if err != nil {
			return err
		}

		err = serversApi.ConfirmResize(d.Id())

		d.SetPartial("flavor_ref")
	}

	d.Partial(false)

	return nil
}

func resourceComputeRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	serversApi, err := p.getServersApi()
	if err != nil {
		return err
	}

	server, err := serversApi.ServerById(d.Id())
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

	// TODO check networks, seucrity groups and floating ip

	d.Set("name", server.Name)
	d.Set("flavor_ref", server.Flavor.Id)

	return nil
}

func WaitForServerState(api gophercloud.CloudServersProvider, id string) resource.StateRefreshFunc {

	return func() (interface{}, string, error) {
		s, err := api.ServerById(id)
		if err != nil {
			httpError, ok := err.(*perigee.UnexpectedResponseCodeError)
			if !ok {
				return nil, "", err
			}

			if httpError.Actual == 404 {
				return s, "DELETED", nil
			}

			return nil, "", err
		}

		return s, s.Status, nil

	}
}
