package openstack

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/blockstorage/v1/volumes"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/volumeattach"
)

func resourceVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceVolumeCreate,
		Read:   resourceVolumeRead,
		Update: resourceVolumeUpdate,
		Delete: resourceVolumeDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"api_version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "1",
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"size": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"volume_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"snapshot_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"source_volume_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"image_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"image_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
				Default:  nil,
			},

			// read-only / exported
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"bootable": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"created_at": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"attach": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"instance_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"device": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
				Set: resourceVolumeAttachmentHash,
			},
		},
	}
}

func resourceVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	client, err := getClient("block", d, meta)
	if err != nil {
		return err
	}

	// figure out the image
	imageID := d.Get("image_id").(string)
	imageName := d.Get("image_name").(string)
	if imageName != "" {
		imageID, err = getImageID(client, d)
		if err != nil {
			return err
		}
	}

	metadata := make(map[string]string)
	if m, ok := d.GetOk("metadata"); ok {
		if len(m.(map[string]interface{})) > 1 {
			for k, v := range m.(map[string]interface{}) {
				metadata[k] = v.(string)
			}
		}
	} else {
		metadata = nil
	}

	opts := &volumes.CreateOpts{
		Availability: d.Get("availability_zone").(string),
		Description:  d.Get("description").(string),
		Metadata:     metadata,
		Name:         d.Get("name").(string),
		Size:         d.Get("size").(int),
		SnapshotID:   d.Get("snapshot_id").(string),
		SourceVolID:  d.Get("source_volume_id").(string),
		ImageID:      imageID,
		VolumeType:   d.Get("volume_type").(string),
	}

	newVolume, err := volumes.Create(client, opts).Extract()
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"BUILD"},
		Target:     "available",
		Refresh:    waitForVolumeState(client, newVolume.ID),
		Timeout:    30 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 2 * time.Second,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return err
	}

	d.SetId(newVolume.ID)

	// were attachments specified?
	if v := d.Get("attach"); v != nil {
		vs := v.(*schema.Set).List()
		if len(vs) > 0 {
			computeClient, err := getClient("compute", d, meta)
			if err != nil {
				return err
			}

			for _, v := range vs {
				va := v.(map[string]interface{})
				if err := checkAttachmentParams(va); err != nil {
					return err
				}

				instanceId := va["instance_id"].(string)
				device := va["device"].(string)

				vaOpts := &volumeattach.CreateOpts{
					Device:   device,
					VolumeID: newVolume.ID,
				}
				if _, err := volumeattach.Create(computeClient, instanceId, vaOpts).Extract(); err != nil {
					return err
				}
			}

			stateConf = &resource.StateChangeConf{
				Pending:    []string{"available", "attaching"},
				Target:     "in-use",
				Refresh:    waitForVolumeState(client, d.Id()),
				Timeout:    30 * time.Minute,
				Delay:      5 * time.Second,
				MinTimeout: 2 * time.Second,
			}

			if _, err := stateConf.WaitForState(); err != nil {
				return err
			}

		}
	}

	if err := setVolumeDetails(client, newVolume.ID, d); err != nil {
		return err
	}

	return nil
}

func resourceVolumeRead(d *schema.ResourceData, meta interface{}) error {
	client, err := getClient("block", d, meta)
	if err != nil {
		return err
	}

	if err := setVolumeDetails(client, d.Id(), d); err != nil {
		return err
	}

	return nil
}

func resourceVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	client, err := getClient("block", d, meta)
	if err != nil {
		return err
	}

	if d.HasChange("attach") {
		oa, na := d.GetChange("attach")
		oas := oa.(*schema.Set).List()
		if len(oas) > 0 {
			computeClient, err := getClient("compute", d, meta)
			if err != nil {
				return err
			}

			// just delete all old attachments
			// kind of sloppy -- this might not be the best way
			for _, v := range oas {
				va := v.(map[string]interface{})
				if err := checkAttachmentParams(va); err != nil {
					return err
				}

				instanceId := va["instance_id"].(string)

				if err := volumeattach.Delete(computeClient, instanceId, d.Id()).ExtractErr(); err != nil {
					return err
				}
			}
		}

		// wait until the volume is in an available state before proceeding
		stateConf := &resource.StateChangeConf{
			Pending:    []string{"in-use", "detaching"},
			Target:     "available",
			Refresh:    waitForVolumeState(client, d.Id()),
			Timeout:    30 * time.Minute,
			Delay:      5 * time.Second,
			MinTimeout: 2 * time.Second,
		}

		if _, err := stateConf.WaitForState(); err != nil {
			return err
		}

		nas := na.(*schema.Set).List()
		if len(nas) > 0 {
			computeClient, err := getClient("compute", d, meta)
			if err != nil {
				return err
			}

			// create all new attachments
			for _, v := range nas {
				va := v.(map[string]interface{})
				if err := checkAttachmentParams(va); err != nil {
					return err
				}

				instanceId := va["instance_id"].(string)
				device := va["device"].(string)

				vaOpts := &volumeattach.CreateOpts{
					Device:   device,
					VolumeID: d.Id(),
				}
				if _, err := volumeattach.Create(computeClient, instanceId, vaOpts).Extract(); err != nil {
					return err
				}
			}

			stateConf = &resource.StateChangeConf{
				Pending:    []string{"available", "attaching"},
				Target:     "in-use",
				Refresh:    waitForVolumeState(client, d.Id()),
				Timeout:    30 * time.Minute,
				Delay:      5 * time.Second,
				MinTimeout: 2 * time.Second,
			}

			if _, err := stateConf.WaitForState(); err != nil {
				return err
			}
		}
	}

	if err := setVolumeDetails(client, d.Id(), d); err != nil {
		return err
	}

	return nil
}

func resourceVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	client, err := getClient("block", d, meta)
	if err != nil {
		return err
	}

	// were attachments specified?
	// if so, detach them before deleting
	if v := d.Get("attach"); v != nil {
		vs := v.(*schema.Set).List()
		if len(vs) > 0 {
			computeClient, err := getClient("compute", d, meta)
			if err != nil {
				return err
			}

			for _, v := range vs {
				va := v.(map[string]interface{})
				if err := checkAttachmentParams(va); err != nil {
					return err
				}

				instanceId := va["instance_id"].(string)

				if err := deleteVolumeAttachment(computeClient, instanceId, d.Id()); err != nil {
					return err
				}
			}
		}
	}

	volume, _ := volumes.Get(client, d.Id()).Extract()
	if err != nil {
		return err
	}

	// wait until the volume is in an available state before proceeding
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"in-use", "detaching"},
		Target:     "available",
		Refresh:    waitForVolumeState(client, d.Id()),
		Timeout:    30 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 2 * time.Second,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return err
	}

	if err := volumes.Delete(client, volume.ID).ExtractErr(); err != nil {
		return err
	}

	stateConf = &resource.StateChangeConf{
		Pending:    []string{"in-use", "detaching"},
		Target:     "DELETED",
		Refresh:    waitForVolumeState(client, d.Id()),
		Timeout:    30 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 2 * time.Second,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return err
	}

	return nil
}

func setVolumeDetails(client *gophercloud.ServiceClient, volumeID string, d *schema.ResourceData) error {
	volume, err := volumes.Get(client, volumeID).Extract()
	if err != nil {
		return err
	}

	log.Printf("[INFO] Volume info: %v", volume)

	d.Set("id", volume.ID)
	d.Set("status", volume.Status)
	d.Set("name", volume.Name)
	d.Set("availability_zone", volume.AvailabilityZone)
	d.Set("bootable", volume.Bootable)
	d.Set("created_at", volume.CreatedAt)
	d.Set("description", volume.Description)
	d.Set("volume_type", volume.VolumeType)
	d.Set("snapshot_id", volume.SnapshotID)
	d.Set("source_volume_id", volume.SourceVolID)
	d.Set("size", volume.Size)
	log.Printf("[INFO] Volume attachments: %v", volume.Attachments)

	if len(volume.Attachments) > 0 {
		attachments := make([]map[string]interface{}, len(volume.Attachments))
		for i, attachment := range volume.Attachments {
			attachments[i] = make(map[string]interface{})
			attachments[i]["id"] = attachment["id"]
			attachments[i]["instance_id"] = attachment["server_id"]
			attachments[i]["device"] = attachment["device"]
		}
		log.Printf("[INFO] Volume attachments: %v", attachments)
		d.Set("attach", attachments)
		log.Printf("[INFO] Volume attachments: %v", d.Get("attach"))
	}

	// TODO: metadata

	return nil
}

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

func resourceVolumeAttachmentHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["instance_id"].(string)))
	return hashcode.String(buf.String())
}

func checkAttachmentParams(va map[string]interface{}) error {
	instanceId := va["instance_id"].(string)

	if instanceId == "" {
		return fmt.Errorf("An instance_id is required for an attachment.")
	}

	return nil
}
