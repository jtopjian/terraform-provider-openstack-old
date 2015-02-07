package openstack

import (
	"bytes"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
)

func resourceSecgroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceSecgroupCreate,
		Read:   resourceSecgroupRead,
		Update: resourceSecgroupUpdate,
		Delete: resourceSecgroupDelete,

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
				Default:  "2",
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"rule": &schema.Schema{
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
						"from_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"to_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"cidr": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"source_group": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"parent_group_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
				Set: resourceSecgroupRuleHash,
			},

			// read-only / exported
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceSecgroupCreate(d *schema.ResourceData, meta interface{}) error {
	client, err := getClient("compute", d, meta)
	if err != nil {
		return err
	}

	secgroupName := d.Get("name").(string)
	secgroupDescription := d.Get("description").(string)

	opts := &secgroups.CreateOpts{
		Name:        secgroupName,
		Description: secgroupDescription,
	}

	newSG, err := secgroups.Create(client, opts).Extract()
	if err != nil {
		return err
	}

	d.SetId(newSG.ID)
	if err := setSecgroupDetails(client, newSG.ID, d); err != nil {
		return err
	}

	// if any rules exist...
	if rs := d.Get("rule").(*schema.Set); rs.Len() > 0 {
		// create an empty schema set to hold all rules
		rules := &schema.Set{
			F: resourceSecgroupRuleHash,
		}

		// loop through each rule and create it
		for _, rule := range rs.List() {
			err := resourceSecgroupRuleCreate(d, meta, newSG.ID, rule.(map[string]interface{}))
			rules.Add(rule)
			d.Set("rule", rules)
			if err != nil {
				return err
			}
		}
	}

	return nil

}

func resourceSecgroupRuleCreate(d *schema.ResourceData, meta interface{}, sgID string, rule map[string]interface{}) error {
	client, err := getClient("compute", d, meta)
	if err != nil {
		return err
	}

	opts := &secgroups.CreateRuleOpts{
		ParentGroupID: sgID,
		FromPort:      rule["from_port"].(int),
		ToPort:        rule["to_port"].(int),
		IPProtocol:    rule["protocol"].(string),
	}

	if rule["cidr"].(string) != "" {
		opts.CIDR = rule["cidr"].(string)
	} else if rule["source_group"].(string) != "" {
		opts.FromGroupID = rule["source_group"].(string)
	} else {
		return fmt.Errorf("At least one of cidr or source_group must be used.")
	}

	newRule, err := secgroups.CreateRule(client, opts).Extract()
	if err != nil {
		return err
	}

	rule["id"] = newRule.ID
	rule["parent_group_id"] = newRule.ParentGroupID

	return nil
}

func resourceSecgroupRead(d *schema.ResourceData, meta interface{}) error {
	client, err := getClient("compute", d, meta)
	if err != nil {
		return err
	}

	if err := setSecgroupDetails(client, d.Id(), d); err != nil {
		return err
	}

	return nil
}

func resourceSecgroupUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("rule") {
		o, n := d.GetChange("rule")
		ors := o.(*schema.Set).Difference(n.(*schema.Set))
		nrs := n.(*schema.Set).Difference(o.(*schema.Set))

		// delete old rules
		for _, rule := range ors.List() {
			if err := resourceSecgroupRuleDelete(d, meta, d.Get("id").(string), rule.(map[string]interface{})); err != nil {
				return err
			}
		}

		// save the state of the current rules
		rules := o.(*schema.Set).Intersection(n.(*schema.Set))
		d.Set("rule", rules)

		// create the new rules
		for _, rule := range nrs.List() {
			err := resourceSecgroupRuleCreate(d, meta, d.Get("id").(string), rule.(map[string]interface{}))
			if err != nil {
				return err
			}
			rules.Add(rule)
			d.Set("rule", rules)
		}

	}

	return nil
}

func resourceSecgroupDelete(d *schema.ResourceData, meta interface{}) error {
	client, err := getClient("compute", d, meta)
	if err != nil {
		return err
	}

	if err := secgroups.Delete(client, d.Id()).ExtractErr(); err != nil {
		return err
	}

	return nil
}

func resourceSecgroupRuleDelete(d *schema.ResourceData, meta interface{}, sgID string, rule map[string]interface{}) error {

	client, err := getClient("compute", d, meta)
	if err != nil {
		return err
	}

	if err := secgroups.DeleteRule(client, rule["id"].(string)).ExtractErr(); err != nil {
		return err
	}

	return nil
}

func setSecgroupDetails(client *gophercloud.ServiceClient, sID string, d *schema.ResourceData) error {
	sg, err := secgroups.Get(client, sID).Extract()
	if err != nil {
		return err
	}
	log.Printf("[INFO] Security Group info: %v", sg)

	d.Set("name", sg.Name)
	d.Set("id", sg.ID)
	d.Set("description", sg.Description)
	d.Set("tenant_id", sg.TenantID)

	return nil
}

func resourceSecgroupRuleHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%d-", m["from_port"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["to_port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["protocol"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cidr"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["source_group"].(string)))
	return hashcode.String(buf.String())
}
