package azurestack

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/2019-03-01/network/mgmt/network"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurestack/azurestack/helpers/utils"
)

func dataSourceArmVirtualNetwork() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceArmVnetRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"address_spaces": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"dns_servers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"subnets": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			// Not supported for 2017-03-09 profile
			// "vnet_peerings": {
			// 	Type:     schema.TypeMap,
			// 	Computed: true,
			// },
		},
	}
}

func dataSourceArmVnetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ArmClient).vnetClient

	resGroup := d.Get("resource_group_name").(string)
	name := d.Get("name").(string)

	resp, err := client.Get(ctx, resGroup, name, "")
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			return diag.Errorf("Error making Read request on Azure virtual network %q (resource group %q): %+v", name, resGroup, err)
		}
		return diag.FromErr(err)
	}

	d.SetId(*resp.ID)

	if props := resp.VirtualNetworkPropertiesFormat; props != nil {
		addressSpaces := flattenVnetAddressPrefixes(props.AddressSpace.AddressPrefixes)
		if err := d.Set("address_spaces", addressSpaces); err != nil {
			return diag.FromErr(err)
		}

		if options := props.DhcpOptions; options != nil {
			dnsServers := flattenVnetAddressPrefixes(options.DNSServers)
			if err := d.Set("dns_servers", dnsServers); err != nil {
				return diag.FromErr(err)
			}
		}

		subnets := flattenVnetSubnetsNames(props.Subnets)
		if err := d.Set("subnets", subnets); err != nil {
			return diag.FromErr(err)
		}

		// Not supported for 2017-03-09 profile
		// vnetPeerings := flattenVnetPeerings(props.VirtualNetworkPeerings)
		// if err := d.Set("vnet_peerings", vnetPeerings); err != nil {
		// 	return diag.FromErr(err)
		// }
	}
	return nil
}

func flattenVnetAddressPrefixes(input *[]string) []interface{} {
	prefixes := make([]interface{}, 0)

	if myprefixes := input; myprefixes != nil {
		for _, prefix := range *myprefixes {
			prefixes = append(prefixes, prefix)
		}
	}
	return prefixes
}

func flattenVnetSubnetsNames(input *[]network.Subnet) []interface{} {
	subnets := make([]interface{}, 0)

	if mysubnets := input; mysubnets != nil {
		for _, subnet := range *mysubnets {
			subnets = append(subnets, *subnet.Name)
		}
	}
	return subnets
}

// Not supported for 2017-03-09 profile
// func flattenVnetPeerings(input *[]network.VirtualNetworkPeering) map[string]interface{} {
// 	output := make(map[string]interface{}, 0)
//
// 	if peerings := input; peerings != nil {
// 		for _, vnetpeering := range *peerings {
// 			key := *vnetpeering.Name
// 			value := *vnetpeering.RemoteVirtualNetwork.ID
//
// 			output[key] = value
//
// 		}
// 	}
// 	return output
// }
