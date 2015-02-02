package openstack

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
)

/*
The following is based on code found in an old gophercloud commit:

https://github.com/rackspace/gophercloud/blob/e13cda260ce48d63ce816f4fa72b6c6cd096596d/floating_ips.go
*/

const novaNetworkFloatingIPResourcePath = "os-floating-ips"

type NovaNetworkFloatingIP struct {
	Id         int    `json:"id"`
	Pool       string `json:"pool"`
	Ip         string `json:"ip"`
	FixedIp    string `json:"fixed_ip"`
	InstanceId string `json:"instance_id"`
}

func listNovaNetworkFloatingIPs(client *gophercloud.ServiceClient) ([]NovaNetworkFloatingIP, error) {
	var fips []NovaNetworkFloatingIP

	_, err := perigee.Request(
		"GET",
		client.ServiceURL(novaNetworkFloatingIPResourcePath),
		perigee.Options{
			MoreHeaders: client.AuthenticatedHeaders(),
			Results: &struct {
				NovaNetworkFloatingIPs *[]NovaNetworkFloatingIP `json:"floating_ips"`
			}{&fips},
		},
	)

	log.Printf("FloatingIP List fips: %v", fips)

	return fips, err
}

func getNovaNetworkFloatingIP(client *gophercloud.ServiceClient, id int) (NovaNetworkFloatingIP, error) {
	var fip NovaNetworkFloatingIP
	_, err := perigee.Request(
		"GET",
		client.ServiceURL(novaNetworkFloatingIPResourcePath, strconv.Itoa(id)),
		perigee.Options{
			MoreHeaders: client.AuthenticatedHeaders(),
			Results: &struct {
				NovaNetworkFloatingIP *NovaNetworkFloatingIP `json:"floating_ip"`
			}{&fip},
		},
	)

	return fip, err
}

func createNovaNetworkFloatingIP(client *gophercloud.ServiceClient, pool string) (NovaNetworkFloatingIP, error) {
	fip := new(NovaNetworkFloatingIP)

	_, err := perigee.Request(
		"POST",
		client.ServiceURL(novaNetworkFloatingIPResourcePath),
		perigee.Options{
			MoreHeaders: client.AuthenticatedHeaders(),
			ReqBody: map[string]string{
				"pool": pool,
			},
			Results: &struct {
				NovaNetworkFloatingIP **NovaNetworkFloatingIP `json:"floating_ip"`
			}{&fip},
		},
	)

	if fip.Ip == "" {
		return *fip, errors.New("Error creating floating IP")
	}

	return *fip, err
}

func deleteNovaNetworkFloatingIP(client *gophercloud.ServiceClient, id int) error {
	_, err := perigee.Request(
		"DELETE",
		client.ServiceURL(novaNetworkFloatingIPResourcePath, strconv.Itoa(id)),
		perigee.Options{
			MoreHeaders: client.AuthenticatedHeaders(),
			OkCodes:     []int{202},
		},
	)

	if err != nil {
		return err
	} else {
		return nil
	}
}

func associateNovaNetworkFloatingIP(client *gophercloud.ServiceClient, serverId string, ip NovaNetworkFloatingIP) error {
	ep := fmt.Sprintf("servers/%s/action", serverId)
	_, err := perigee.Request(
		"POST",
		client.ServiceURL(ep),
		perigee.Options{
			MoreHeaders: client.AuthenticatedHeaders(),
			ReqBody: map[string](map[string]string){
				"addFloatingIp": map[string]string{"address": ip.Ip},
			},
			OkCodes: []int{202},
		},
	)

	if err != nil {
		return err
	} else {
		return nil
	}
}

func disassociateNovaNetworkFloatingIP(client *gophercloud.ServiceClient, serverId string, ip NovaNetworkFloatingIP) error {
	ep := fmt.Sprintf("servers/%s/action", serverId)
	_, err := perigee.Request(
		"POST",
		client.ServiceURL(ep),
		perigee.Options{
			MoreHeaders: client.AuthenticatedHeaders(),
			ReqBody: map[string](map[string]string){
				"removeFloatingIp": map[string]string{"address": ip.Ip},
			},
			OkCodes: []int{202},
		},
	)

	if err != nil {
		return err
	} else {
		return nil
	}
}

func setNovaNetworkFloatingIPDetails(client *gophercloud.ServiceClient, id int, d *schema.ResourceData) error {
	fip, err := getNovaNetworkFloatingIP(client, id)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Floating IP info: %v", fip)

	d.Set("pool", fip.Pool)
	d.Set("ip", fip.Ip)
	d.Set("fixed_ip", fip.FixedIp)
	d.Set("instance_id", fip.InstanceId)

	return nil
}
