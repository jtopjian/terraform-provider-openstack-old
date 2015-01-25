package openstack

import (
	"fmt"
	"testing"

	"github.com/haklop/gophercloud-extensions/network"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
)

func TestAccOpenstackNetwork(t *testing.T) {
	var network network.Network

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOpenstackNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testNetworkConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOpenstackNetworkExists(
						"openstack_network.accept_test", &network),
				),
			},
		},
	})
}

func testAccCheckOpenstackNetworkDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_network" {
			continue
		}

		networksApi, err := network.NetworksApi(config.AccessProvider, gophercloud.ApiCriteria{
			Name:      "neutron",
			UrlChoice: gophercloud.PublicURL,
		})
		if err != nil {
			return err
		}

		_, err = networksApi.GetNetwork(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Network (%s) still exists.", rs.Primary.ID)
		}

		httpError, ok := err.(*perigee.UnexpectedResponseCodeError)
		if !ok {
			return fmt.Errorf("Unkonw Security Group error")
		}

		if httpError.Actual != 404 {
			return httpError
		}
	}

	return nil
}

func testAccCheckOpenstackNetworkExists(n string, net *network.Network) resource.TestCheckFunc {

	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		networksApi, err := network.NetworksApi(config.AccessProvider, gophercloud.ApiCriteria{
			Name:      "neutron",
			UrlChoice: gophercloud.PublicURL,
		})
		if err != nil {
			return err
		}

		found, err := networksApi.GetNetwork(rs.Primary.ID)

		if found.Id != rs.Primary.ID {
			return fmt.Errorf("Network not found")
		}

		*net = *found

		return nil
	}
}

const testNetworkConfig = `
resource "openstack_network" "accept_test" {
    name = "accept_test Network"
}
`
