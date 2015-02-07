package openstack

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/flavors"
	"github.com/rackspace/gophercloud/openstack/compute/v2/images"
	"github.com/rackspace/gophercloud/pagination"
)

// images
func getImageID(client *gophercloud.ServiceClient, d *schema.ResourceData) (string, error) {
	imageID := d.Get("image_id").(string)
	imageName := d.Get("image_name").(string)
	if imageID == "" {
		pager := images.ListDetail(client, nil)

		pager.EachPage(func(page pagination.Page) (bool, error) {
			imageList, err := images.ExtractImages(page)

			if err != nil {
				return false, err
			}

			for _, i := range imageList {
				if i.Name == imageName {
					imageID = i.ID
				}
			}
			return true, nil
		})

		if imageID == "" {
			return "", fmt.Errorf("Unable to find image: %v", imageName)
		}
	}

	if imageID == "" && imageName == "" {
		return "", fmt.Errorf("Neither an image ID nor an image name were able to be determined.")
	}

	return imageID, nil
}

func getImage(client *gophercloud.ServiceClient, imageId string) (*images.Image, error) {
	return images.Get(client, imageId).Extract()
}

// flavors
func getFlavorID(client *gophercloud.ServiceClient, d *schema.ResourceData) (string, error) {
	flavorID := d.Get("flavor_id").(string)
	flavorName := d.Get("flavor_name").(string)
	if flavorID == "" {
		pager := flavors.ListDetail(client, nil)

		pager.EachPage(func(page pagination.Page) (bool, error) {
			flavorList, err := flavors.ExtractFlavors(page)

			if err != nil {
				return false, err
			}

			for _, f := range flavorList {
				if f.Name == flavorName {
					flavorID = f.ID
				}
			}
			return true, nil
		})

		if flavorID == "" {
			return "", fmt.Errorf("Unable to find flavor: %v", flavorName)
		}
	}

	if flavorID == "" && flavorName == "" {
		return "", fmt.Errorf("Neither a flavor ID nor a flavor name were able to be determined.")
	}

	return flavorID, nil
}

func getFlavor(client *gophercloud.ServiceClient, flavorId string) (*flavors.Flavor, error) {
	return flavors.Get(client, flavorId).Extract()
}
