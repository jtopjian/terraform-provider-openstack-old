package openstack

import (
	"errors"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/flavors"
	"github.com/rackspace/gophercloud/pagination"
)

func GetFlavorIDByName(client *gophercloud.ServiceClient, flavorName string) (string, error) {
	pager := flavors.ListDetail(client, nil)

	flavorID := ""

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
		return "", errors.New("Unable to find flavor")
	} else {
		return flavorID, nil
	}
}
