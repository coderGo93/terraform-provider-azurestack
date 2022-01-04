package azurestack

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/profiles/2017-03-09/resources/mgmt/resources"
	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurestack/azurestack/helpers/utils"
)

func resourceArmResourceGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceArmResourceGroupCreateUpdate,
		ReadContext:   resourceArmResourceGroupRead,
		UpdateContext: resourceArmResourceGroupCreateUpdate,
		DeleteContext: resourceArmResourceGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": resourceGroupNameSchema(),

			"location": locationSchema(),

			"tags": tagsSchema(),
		},
	}
}

func resourceArmResourceGroupCreateUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ArmClient).resourceGroupsClient

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	tags := d.Get("tags").(map[string]interface{})
	parameters := resources.Group{
		Location: utils.String(location),
		Tags:     *expandTags(tags),
	}
	_, err := client.CreateOrUpdate(ctx, name, parameters)
	if err != nil {
		return diag.Errorf("Error creating resource group: %+v", err)
	}

	resp, err := client.Get(ctx, name)
	if err != nil {
		return diag.Errorf("Error retrieving resource group: %+v", err)
	}

	d.SetId(*resp.ID)

	return resourceArmResourceGroupRead(ctx, d, meta)
}

func resourceArmResourceGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ArmClient).resourceGroupsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return diag.Errorf("Error parsing Azure Resource ID %q: %+v", d.Id(), err)
	}

	name := id.ResourceGroup

	resp, err := client.Get(ctx, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[INFO] Error reading resource group %q - removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return diag.Errorf("Error reading resource group: %+v", err)
	}

	d.Set("name", resp.Name)
	d.Set("location", azureStackNormalizeLocation(*resp.Location))
	flattenAndSetTags(d, &resp.Tags)

	return nil
}

func resourceArmResourceGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ArmClient).resourceGroupsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return diag.Errorf("Error parsing Azure Resource ID %q: %+v", d.Id(), err)
	}

	name := id.ResourceGroup

	deleteFuture, err := client.Delete(ctx, name)
	if err != nil {
		if response.WasNotFound(deleteFuture.Response()) {
			return nil
		}

		return diag.Errorf("Error deleting Resource Group %q: %+v", name, err)
	}

	err = deleteFuture.WaitForCompletionRef(ctx, client.Client)
	if err != nil {
		if response.WasNotFound(deleteFuture.Response()) {
			return nil
		}

		return diag.Errorf("Error deleting Resource Group %q: %+v", name, err)
	}

	return nil
}
