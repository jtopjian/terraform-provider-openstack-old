package openstack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/keypairs"
)

const public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAYQCvwx+DisBkZmOsZEKhiBogTahL4Wqv2OwNjjrX+Ft7li7nDrUDeWCFTgwd3dlJkMx+jYxLBNeJ/aJkChJhKalr8ZzEQID1d2b7TbAszrLGrO9GeGy1bQt28AxNeRzBiwM= terraform_test"

func TestAccComputeV2Keypair(t *testing.T) {
	var keypair keypairs.KeyPair

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeV2KeypairDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeV2Keypair,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2KeypairExists(
						t, "openstack_keypair.accept_test", &keypair),
					resource.TestCheckResourceAttr(
						"openstack_keypair.accept_test", "name", "accept_test"),
					resource.TestCheckResourceAttr(
						"openstack_keypair.accept_test", "public_key", public_key),
				),
			},
		},
	})
}

func testComputeClient() (*gophercloud.ServiceClient, error) {
	config := testAccProvider.Meta().(*Config)
	computeClient, err := config.computeClient(OS_REGION_NAME, "2")
	if err != nil {
		return nil, fmt.Errorf("Problem getting client: ", err)
	}
	return computeClient, nil
}

func testAccCheckComputeV2KeypairExists(t *testing.T, n string, keypair *keypairs.KeyPair) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %v", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		computeClient, err := testComputeClient()
		if err != nil {
			return err
		}

		found, err := keypairs.Get(computeClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Keypair not found: %v", found.Name)
		}

		*keypair = *found

		return nil
	}
}

func testAccCheckComputeV2KeypairDestroy(s *terraform.State) error {
	computeClient, err := testComputeClient()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_keypair" {
			continue
		}

		_, err := keypairs.Get(computeClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Keypair still exists.")
		}
	}

	return nil
}

var testAccComputeV2Keypair = fmt.Sprintf(`
	resource "openstack_keypair" "accept_test" {
		region = "%s"
		name = "accept_test"
		public_key = "%s"
	}`,
	OS_REGION_NAME, public_key,
)
