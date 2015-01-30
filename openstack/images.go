package openstack

import (
	"errors"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/images"
	"github.com/rackspace/gophercloud/pagination"
)

func GetImageIDByName(client *gophercloud.ServiceClient, imageName string) (string, error) {
	pager := images.ListDetail(client, nil)

	imageID := ""

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
		return "", errors.New("Unable to find image")
	} else {
		return imageID, nil
	}
}
