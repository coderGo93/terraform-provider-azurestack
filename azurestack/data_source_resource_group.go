package azurestack

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceArmResourceGroup() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceArmResourceGroupRead,

		Schema: map[string]*schema.Schema{
			"name":     resourceGroupNameForDataSourceSchema(),
			"location": locationForDataSourceSchema(),
			"tags":     tagsForDataSourceSchema(),
		},
	}
}

func dataSourceArmResourceGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ArmClient).resourceGroupsClient

	name := d.Get("name").(string)
	resp, err := client.Get(ctx, name)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*resp.ID)

	return resourceArmResourceGroupRead(ctx, d, meta)
}
