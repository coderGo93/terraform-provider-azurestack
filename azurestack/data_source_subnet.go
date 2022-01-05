package azurestack

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurestack/azurestack/helpers/utils"
)

func dataSourceArmSubnet() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceArmSubnetRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},

			"virtual_network_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},

			"resource_group_name": resourceGroupNameForDataSourceSchema(),

			"address_prefix": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"network_security_group_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"route_table_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ip_configurations": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func dataSourceArmSubnetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ArmClient).subnetClient

	name := d.Get("name").(string)
	virtualNetworkName := d.Get("virtual_network_name").(string)
	resourceGroup := d.Get("resource_group_name").(string)

	resp, err := client.Get(ctx, resourceGroup, virtualNetworkName, name, "")
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			return diag.Errorf("Error: Subnet %q (Virtual Network %q / Resource Group %q) was not found", name, resourceGroup, virtualNetworkName)
		}
		return diag.Errorf("Error making Read request on Azure Subnet %q: %+v", name, err)
	}
	d.SetId(*resp.ID)

	d.Set("name", name)
	d.Set("resource_group_name", resourceGroup)
	d.Set("virtual_network_name", virtualNetworkName)

	if props := resp.SubnetPropertiesFormat; props != nil {
		d.Set("address_prefix", props.AddressPrefix)

		if props.NetworkSecurityGroup != nil {
			d.Set("network_security_group_id", props.NetworkSecurityGroup.ID)
		} else {
			d.Set("network_security_group_id", "")
		}

		if props.RouteTable != nil {
			d.Set("route_table_id", props.RouteTable.ID)
		} else {
			d.Set("route_table_id", "")
		}

		if err := d.Set("ip_configurations", flattenSubnetIPConfigurations(props.IPConfigurations)); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
