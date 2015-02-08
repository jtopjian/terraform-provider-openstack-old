package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/blockstorage/v1/volumes"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/volumeattach"
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

// volumes
func waitForVolumeState(client *gophercloud.ServiceClient, volumeId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		latest, err := volumes.Get(client, volumeId).Extract()
		if err != nil {
			httpStatus := err.(*perigee.UnexpectedResponseCodeError)
			if httpStatus.Actual == 404 {
				return "", "DELETED", nil
			}
			return nil, "", err
		}

		log.Printf("Volume status: %v", latest.Status)
		return latest, latest.Status, nil
	}
}

func attachVolumes(computeClient *gophercloud.ServiceClient, blockClient *gophercloud.ServiceClient, serverId string, vols []interface{}) error {
	if len(vols) > 0 {
		for _, v := range vols {
			va := v.(map[string]interface{})
			volumeId := va["volume_id"].(string)
			device := va["device"].(string)

			s := ""
			if serverId != "" {
				s = serverId
			} else if va["server_id"] != "" {
				s = va["server_id"].(string)
			} else {
				return fmt.Errorf("Unable to determine server ID to attach volume.")
			}

			vaOpts := &volumeattach.CreateOpts{
				Device:   device,
				VolumeID: volumeId,
			}

			if _, err := volumeattach.Create(computeClient, s, vaOpts).Extract(); err != nil {
				return err
			}

			stateConf := &resource.StateChangeConf{
				Pending:    []string{"available", "attaching"},
				Target:     "in-use",
				Refresh:    waitForVolumeState(blockClient, va["volume_id"].(string)),
				Timeout:    30 * time.Minute,
				Delay:      5 * time.Second,
				MinTimeout: 2 * time.Second,
			}

			if _, err := stateConf.WaitForState(); err != nil {
				return err
			}
		}
	}
	return nil
}

func detachVolumes(computeClient *gophercloud.ServiceClient, blockClient *gophercloud.ServiceClient, serverId string, vols []interface{}) error {
	if len(vols) > 0 {
		for _, v := range vols {
			va := v.(map[string]interface{})
			aId := va["id"].(string)

			s := ""
			if serverId != "" {
				s = serverId
			} else if va["server_id"] != "" {
				s = va["server_id"].(string)
			} else {
				return fmt.Errorf("Unable to determine server ID to detach volume.")
			}

			if err := volumeattach.Delete(computeClient, s, aId).ExtractErr(); err != nil {
				return err
			}

			stateConf := &resource.StateChangeConf{
				Pending:    []string{"in-use", "detaching"},
				Target:     "available",
				Refresh:    waitForVolumeState(blockClient, va["volume_id"].(string)),
				Timeout:    30 * time.Minute,
				Delay:      5 * time.Second,
				MinTimeout: 2 * time.Second,
			}

			if _, err := stateConf.WaitForState(); err != nil {
				return err
			}
		}
	}

	return nil
}

func getVolumeAttachments(computeClient *gophercloud.ServiceClient, serverId string) ([]volumeattach.VolumeAttachment, error) {
	var attachments []volumeattach.VolumeAttachment
	err := volumeattach.List(computeClient, serverId).EachPage(func(page pagination.Page) (bool, error) {
		actual, err := volumeattach.ExtractVolumeAttachments(page)
		if err != nil {
			return false, err
		}

		attachments = actual
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return attachments, nil
}
