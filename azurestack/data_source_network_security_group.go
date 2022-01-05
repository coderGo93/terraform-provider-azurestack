package azurestack

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurestack/azurestack/helpers/utils"
)

func dataSourceArmNetworkSecurityGroup() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceArmNetworkSecurityGroupRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"resource_group_name": resourceGroupNameForDataSourceSchema(),

			"location": locationForDataSourceSchema(),

			"security_rule": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"protocol": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"source_port_range": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"source_port_ranges": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"destination_port_range": {
							Type:     schema.TypeString,
							Computed: true,
						},

						// The Following attributes are not included in the profile  2017-03-09
						// destination_port_ranges
						// source_address_prefixes
						// source_application_security_group_ids
						// destination_address_prefixes
						// destination_application_security_group_ids

						// "destination_port_ranges": {
						// 	Type:     schema.TypeSet,
						// 	Computed: true,
						// 	Elem:     &schema.Schema{Type: schema.TypeString},
						// 	Set:      schema.HashString,
						// },

						"source_address_prefix": {
							Type:     schema.TypeString,
							Computed: true,
						},

						// "source_address_prefixes": {
						// 	Type:     schema.TypeSet,
						// 	Computed: true,
						// 	Elem:     &schema.Schema{Type: schema.TypeString},
						// 	Set:      schema.HashString,
						// },

						// "source_application_security_group_ids": {
						// 	Type:     schema.TypeSet,
						// 	Optional: true,
						// 	Elem:     &schema.Schema{Type: schema.TypeString},
						// 	Set:      schema.HashString,
						// },

						"destination_address_prefix": {
							Type:     schema.TypeString,
							Computed: true,
						},

						// "destination_address_prefixes": {
						// 	Type:     schema.TypeSet,
						// 	Computed: true,
						// 	Elem:     &schema.Schema{Type: schema.TypeString},
						// 	Set:      schema.HashString,
						// },

						// "destination_application_security_group_ids": {
						// 	Type:     schema.TypeSet,
						// 	Optional: true,
						// 	Elem:     &schema.Schema{Type: schema.TypeString},
						// 	Set:      schema.HashString,
						// },

						"access": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"priority": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"direction": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"tags": tagsForDataSourceSchema(),
		},
	}
}

func dataSourceArmNetworkSecurityGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ArmClient).secGroupClient

	resourceGroup := d.Get("resource_group_name").(string)
	name := d.Get("name").(string)

	resp, err := client.Get(ctx, resourceGroup, name, "")
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
		}
		return diag.Errorf("Error making Read request on Network Security Group %q (Resource Group %q): %+v", name, resourceGroup, err)
	}

	d.SetId(*resp.ID)

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resourceGroup)
	if location := resp.Location; location != nil {
		d.Set("location", azureStackNormalizeLocation(*location))
	}

	if props := resp.SecurityGroupPropertiesFormat; props != nil {
		flattenedRules := flattenNetworkSecurityRules(props.SecurityRules)
		if err := d.Set("security_rule", flattenedRules); err != nil {
			return diag.Errorf("Error flattening `security_rule`: %+v", err)
		}
	}

	flattenAndSetTags(d, &resp.Tags)

	return nil
}
