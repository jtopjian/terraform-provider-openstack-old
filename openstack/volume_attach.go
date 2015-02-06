package openstack

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
)

type VolumeAttachment struct {
	Id         string `json:"id"`
	Device     string `json:"device"`
	InstanceId string `json:"serverId"`
	VolumeId   string `json:"volumeId"`
}

func listVolumeAttachments(client *gophercloud.ServiceClient, instanceId string) ([]VolumeAttachment, error) {
	var vas []VolumeAttachment
	ep := fmt.Sprintf("servers/%s/os-volume_attachments", instanceId)

	_, err := perigee.Request(
		"GET",
		client.ServiceURL(ep),
		perigee.Options{
			MoreHeaders: client.AuthenticatedHeaders(),
			Results: &struct {
				VolumeAttachments *[]VolumeAttachment `json:"volumeAttachments"`
			}{&vas},
		},
	)

	log.Printf("[INFO] Volume Attachments: %v", vas)

	return vas, err
}

func getVolumeAttachment(client *gophercloud.ServiceClient, instanceId, vaId string) (VolumeAttachment, error) {
	var va VolumeAttachment
	ep := fmt.Sprintf("servers/%s/os-volume_attachments/%s", instanceId, vaId)

	_, err := perigee.Request(
		"GET",
		client.ServiceURL(ep),
		perigee.Options{
			MoreHeaders: client.AuthenticatedHeaders(),
			Results: &struct {
				VolumeAttachment *VolumeAttachment `json:"volumeAttachment"`
			}{&va},
		},
	)

	return va, err
}

func createVolumeAttachment(client *gophercloud.ServiceClient, instanceId, volumeId, device string) (VolumeAttachment, error) {
	va := new(VolumeAttachment)
	ep := fmt.Sprintf("servers/%s/os-volume_attachments", instanceId)

	x, err := perigee.Request(
		"POST",
		client.ServiceURL(ep),
		perigee.Options{
			MoreHeaders: client.AuthenticatedHeaders(),
			ReqBody: map[string](map[string]string){
				"volumeAttachment": map[string]string{
					"volumeId": volumeId,
					"device":   device,
				},
			},
			Results: &struct {
				VolumeAttachment **VolumeAttachment `json:"volumeAttachment"`
			}{&va},
		},
	)

	if err != nil {
		return *va, err
	}

	if va.Id == "" {
		return *va, fmt.Errorf("Error attaching volume: %v", x)
	}

	return *va, err
}

func deleteVolumeAttachment(client *gophercloud.ServiceClient, instanceId, volumeId string) error {
	ep := fmt.Sprintf("servers/%s/os-volume_attachments/%s", instanceId, volumeId)
	_, err := perigee.Request(
		"DELETE",
		client.ServiceURL(ep),
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

func setVolumeAttachmentDetails(client *gophercloud.ServiceClient, d *schema.ResourceData, instanceId, vaId string) error {
	va, err := getVolumeAttachment(client, instanceId, vaId)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Volume Attachment info: %v", va)

	d.Set("id", va.Id)
	d.Set("device", va.Device)
	d.Set("volume_id", va.VolumeId)
	d.Set("instance_id", va.InstanceId)

	return nil
}
