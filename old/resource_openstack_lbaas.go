package openstack

import (
	"fmt"
	"log"

	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
)

func resourceLBaaS() *schema.Resource {
	return &schema.Resource{
		Create: resourceLBaaSCreate,
		Read:   resourceLBaaSRead,
		Update: resourceLBaaSUpdate,
		Delete: resourceLBaaSDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"provider": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"lb_method": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"vip_protocol_port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"floating_ip_pool_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"floating_ip": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"member": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"instance_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"member_id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"monitor": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"delay": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"timeout": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"max_retries": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"expected_codes": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"http_method": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"monitor_id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func resourceLBaaSCreate(d *schema.ResourceData, meta interface{}) error {

	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	pool, err := networksApi.CreatePool(network.NewPool{
		Name:        d.Get("name").(string),
		SubnetId:    d.Get("subnet_id").(string),
		LoadMethod:  d.Get("lb_method").(string),
		Protocol:    d.Get("protocol").(string),
		Description: d.Get("description").(string),
		Provider:    d.Get("provider").(string),
	})

	if err != nil {
		return err
	}

	d.SetId(pool.Id)

	membersCount := d.Get("member.#").(int)
	members := make([]*poolMember, 0, membersCount)
	for i := 0; i < membersCount; i++ {
		prefix := fmt.Sprintf("member.%d", i)

		var member poolMember
		member.ProtocolPort = d.Get(prefix + ".port").(int)
		member.InstanceId = d.Get(prefix + ".instance_id").(string)

		members = append(members, &member)
	}

	for _, member := range members {
		// TODO order ports
		ports, err := networksApi.GetPorts()
		if err != nil {
			return err
		}

		var address string
		for _, port := range ports {
			if port.DeviceId == member.InstanceId {
				for _, ips := range port.FixedIps {
					// if possible, select a port on pool subnet
					if ips.SubnetId == d.Get("subnet_id").(string) || address == "" {
						address = ips.IpAddress
					}
				}
			}
		}

		newMember := network.NewMember{
			ProtocolPort: member.ProtocolPort,
			PoolId:       d.Id(),
			AdminStateUp: true,
			Address:      address,
		}

		result, err := networksApi.CreateMember(newMember)
		if err != nil {
			return err
		}
		member.MemberId = result.Id

		// TODO save memberId
	}

	monitorsCount := d.Get("monitor.#").(int)
	for i := 0; i < monitorsCount; i++ {
		prefix := fmt.Sprintf("monitor.%d", i)

		var monitor network.NewMonitor
		monitor.Type = d.Get(prefix + ".type").(string)
		monitor.Delay = d.Get(prefix + ".delay").(int)
		monitor.Timeout = d.Get(prefix + ".timeout").(int)
		monitor.MaxRetries = d.Get(prefix + ".max_retries").(int)
		monitor.ExpectedCodes = d.Get(prefix + ".expected_codes").(string)
		monitor.HttpMethod = d.Get(prefix + ".http_method").(string)

		result, err := networksApi.CreateMonitor(monitor)
		if err != nil {
			return err
		}

		log.Printf("monitor: %#v", result)

		err = networksApi.AssociateMonitor(result.Id, d.Id())
		if err != nil {
			return err
		}
		// TODO save monitor id

	}

	vipProtocolPort, ok := d.GetOk("vip_protocol_port")
	if !ok {
		return nil
	}

	vip, err := networksApi.CreateVip(network.NewVip{
		Name:         d.Get("name").(string) + "_vip",
		Protocol:     d.Get("protocol").(string),
		SubnetId:     d.Get("subnet_id").(string),
		ProtocolPort: vipProtocolPort.(int),
		PoolId:       d.Id(),
	})

	if err != nil {
		return err
	}

	// TODO retrieve pool_id corresponding to the pool name
	_, ok = d.GetOk("floating_ip_pool_id")
	if !ok {
		return nil
	}

	floatingIps, err := networksApi.ListFloatingIps()
	if err != nil {
		return err
	}

	var newIp network.FloatingIp
	hasFloatingIps := false

	for _, element := range floatingIps {
		// TODO check the pool id
		// use first floating ip available on the pool
		log.Printf("ips: %#v", element)
		if len(element.PortId) == 0 {
			newIp = element
			hasFloatingIps = true
		}
	}

	// if there is no available floating ips, try to create a new one
	// FIXME create floatingIp with neutron
	/*if !hasFloatingIps {
		newIp, err = serversApi.CreateFloatingIp(floatingIpPoolId.(string))
		if err != nil {
			return err
		}
	}
	*/

	if hasFloatingIps {
		log.Printf("associate %#v with  %#v", vip, newIp)
		err = networksApi.AssociateFloatingIp(vip.PortId, newIp.Id)
		if err != nil {
			return err
		}

		d.Set("floating_ip", newIp.FloatingIpAddress)

		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Unable to associate a floating ip")
	}

	return nil

}

func resourceLBaaSDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	pool, err := networksApi.GetPool(d.Id())
	if err != nil {
		return err
	}

	for _, member := range pool.Members {
		err = networksApi.DeleteMember(member)
		if err != nil {
			return err
		}
	}

	for _, monitor := range pool.HealthMonitors {

		err = networksApi.UnassociateMonitor(monitor, d.Id())
		if err != nil {
			return err
		}

		err = networksApi.DeleteMonitor(monitor)
		if err != nil {
			return err
		}
	}

	if len(pool.VipId) > 0 {
		err = networksApi.DeleteVip(pool.VipId)
		if err != nil {
			return err
		}
	}

	return networksApi.DeletePool(d.Id())

}

func resourceLBaaSRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	pool, err := networksApi.GetPool(d.Id())
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

	d.Set("name", pool.Name)
	d.Set("description", pool.Description)
	d.Set("lb_method", pool.LoadMethod)

	// TODO compare pool.Members and pool.HealthMonitors

	return nil

}

func resourceLBaaSUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*Config)
	networksApi, err := p.getNetworkApi()
	if err != nil {
		return err
	}

	updatedPool := network.Pool{
		Id: d.Id(),
	}

	if d.HasChange("name") {
		updatedPool.Name = d.Get("name").(string)
	}

	if d.HasChange("lb_method") {
		updatedPool.LoadMethod = d.Get("lb_method").(string)
	}

	if d.HasChange("description") {
		updatedPool.Description = d.Get("description").(string)
	}

	_, err = networksApi.UpdatePool(updatedPool)

	// TODO update members and HealthMonitors

	return err

}

type poolMember struct {
	ProtocolPort int
	InstanceId   string
	MemberId     string
}
