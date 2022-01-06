package azurestack

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-11-01/network"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurestack/azurestack/helpers/utils"
)

func dataSourceArmRouteTable() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceArmRouteTableRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},

			"resource_group_name": resourceGroupNameForDataSourceSchema(),

			"location": locationForDataSourceSchema(),

			"route": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"address_prefix": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"next_hop_type": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"next_hop_in_ip_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"subnets": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"tags": tagsForDataSourceSchema(),
		},
	}
}

func dataSourceArmRouteTableRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ArmClient).routeTablesClient

	name := d.Get("name").(string)
	resourceGroup := d.Get("resource_group_name").(string)

	resp, err := client.Get(ctx, resourceGroup, name, "")
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			return diag.Errorf("Error: Route Table %q (Resource Group %q) was not found", name, resourceGroup)
		}
		return diag.Errorf("Error making Read request on Azure Route Table %q: %+v", name, err)
	}

	d.SetId(*resp.ID)

	d.Set("name", name)
	d.Set("resource_group_name", resourceGroup)
	if location := resp.Location; location != nil {
		d.Set("location", azureStackNormalizeLocation(*location))
	}

	if props := resp.RouteTablePropertiesFormat; props != nil {
		if err := d.Set("route", flattenRouteTableDataSourceRoutes(props.Routes)); err != nil {
			return diag.FromErr(err)
		}

		if err := d.Set("subnets", flattenRouteTableDataSourceSubnets(props.Subnets)); err != nil {
			return diag.FromErr(err)
		}
	}

	flattenAndSetTags(d, &resp.Tags)

	return nil
}

func flattenRouteTableDataSourceRoutes(input *[]network.Route) []interface{} {
	results := make([]interface{}, 0)

	if routes := input; routes != nil {
		for _, route := range *routes {
			r := make(map[string]interface{})

			r["name"] = *route.Name

			if props := route.RoutePropertiesFormat; props != nil {
				r["address_prefix"] = *props.AddressPrefix
				r["next_hop_type"] = string(props.NextHopType)
				if ip := props.NextHopIPAddress; ip != nil {
					r["next_hop_in_ip_address"] = *ip
				}
			}

			results = append(results, r)
		}
	}

	return results
}

func flattenRouteTableDataSourceSubnets(subnets *[]network.Subnet) []string {
	output := make([]string, 0)

	if subnets != nil {
		for _, subnet := range *subnets {
			output = append(output, *subnet.ID)
		}
	}

	return output
}
