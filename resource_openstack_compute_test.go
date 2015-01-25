package openstack

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
)

func TestAccOpenstackCompute(t *testing.T) {
	var server gophercloud.NewServer

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckOpenstackComputeDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testComputeConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOpenstackComputeExists("openstack_compute.accept_test", &server),
					resource.TestCheckResourceAttr(
						"openstack_compute.accept_test", "name", "accept_test_compute"),
					resource.TestCheckResourceAttr(
						"openstack_compute.accept_test", "image_ref", os.Getenv("OS_AT_DEFAULT_IMAGE_REF")),
					resource.TestCheckResourceAttr(
						"openstack_compute.accept_test", "flavor_ref", os.Getenv("OS_AT_DEFAULT_FLAVOR_REF")),
					resource.TestCheckResourceAttr(
						"openstack_compute.accept_test", "user_data",
						"c29tZSBzYW1wbGUgdXNlciBkYXRh"), // base64 encoded user_data
				),
			},
		},
	})
}

func testAccCheckOpenstackComputeDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_compute" {
			continue
		}

		serversApi, err := gophercloud.ServersApi(config.AccessProvider, gophercloud.ApiCriteria{
			Name:      "nova",
			UrlChoice: gophercloud.PublicURL,
		})
		if err != nil {
			return err
		}

		_, err = serversApi.ServerById(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Instance (%s) still exists.", rs.Primary.ID)
		}

		httpError, ok := err.(*perigee.UnexpectedResponseCodeError)
		if !ok {
			return fmt.Errorf("Unkonw computeerror")
		}

		if httpError.Actual != 404 {
			return httpError
		}
	}

	return nil
}

func testAccCheckOpenstackComputeExists(n string, server *gophercloud.NewServer) resource.TestCheckFunc {

	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		serversApi, err := gophercloud.ServersApi(config.AccessProvider, gophercloud.ApiCriteria{
			Name:      "nova",
			UrlChoice: gophercloud.PublicURL,
		})
		if err != nil {
			return err
		}

		_, err = serversApi.ServerById(rs.Primary.ID)
		return err
	}
}

func testComputeConfig() string {
	var conf = "resource \"openstack_compute\" \"accept_test\" {\n"
	conf += "name = \"accept_test_compute\"\n"
	conf += "image_ref = \"" + os.Getenv("OS_AT_DEFAULT_IMAGE_REF") + "\"\n"
	conf += "flavor_ref = \"" + os.Getenv("OS_AT_DEFAULT_FLAVOR_REF") + "\"\n"
	conf += "user_data = \"some sample user data\"\n"
	conf += "}"
	return conf
}
