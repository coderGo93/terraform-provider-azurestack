package azurestack

import (
	"context"
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/compute/mgmt/compute"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurestack/azurestack/helpers/utils"
)

func resourceArmAvailabilitySet() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceArmAvailabilitySetCreate,
		ReadContext:   resourceArmAvailabilitySetRead,
		UpdateContext: resourceArmAvailabilitySetCreate,
		DeleteContext: resourceArmAvailabilitySetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": resourceGroupNameSchema(),

			"location": locationSchema(),

			"platform_update_domain_count": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      5,
				ForceNew:     true,
				ValidateFunc: validation.IntBetween(1, 20),
			},

			"platform_fault_domain_count": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      3,
				ForceNew:     true,
				ValidateFunc: validation.IntBetween(1, 3),
			},

			"managed": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmAvailabilitySetCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ArmClient).availSetClient

	log.Printf("[INFO] preparing arguments for AzureStack Availability Set creation.")

	name := d.Get("name").(string)
	location := azureStackNormalizeLocation(d.Get("location").(string))
	resGroup := d.Get("resource_group_name").(string)
	updateDomainCount := d.Get("platform_update_domain_count").(int)
	faultDomainCount := d.Get("platform_fault_domain_count").(int)

	managed := d.Get("managed").(bool)

	tags := d.Get("tags").(map[string]interface{})

	availSet := compute.AvailabilitySet{
		Name:     &name,
		Location: &location,
		AvailabilitySetProperties: &compute.AvailabilitySetProperties{
			PlatformFaultDomainCount:  utils.Int32(int32(faultDomainCount)),
			PlatformUpdateDomainCount: utils.Int32(int32(updateDomainCount)),
		},
		Tags: *expandTags(tags),
	}

	if managed {
		n := "Aligned"
		availSet.Sku = &compute.Sku{
			Name: &n,
		}
	}

	resp, err := client.CreateOrUpdate(ctx, resGroup, name, availSet)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*resp.ID)

	return resourceArmAvailabilitySetRead(ctx, d, meta)
}

func resourceArmAvailabilitySetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ArmClient).availSetClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	resGroup := id.ResourceGroup
	name := id.Path["availabilitySets"]

	resp, err := client.Get(ctx, resGroup, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("Error making Read request on Azure Availability Set %q (Resource Group %q): %+v", name, resGroup, err)
	}

	availSet := *resp.AvailabilitySetProperties
	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	if location := resp.Location; location != nil {
		d.Set("location", azureStackNormalizeLocation(*location))
	}
	d.Set("platform_update_domain_count", availSet.PlatformUpdateDomainCount)
	d.Set("platform_fault_domain_count", availSet.PlatformFaultDomainCount)

	if resp.Sku != nil && resp.Sku.Name != nil {
		d.Set("managed", strings.EqualFold(*resp.Sku.Name, "Aligned"))
	}

	flattenAndSetTags(d, &resp.Tags)

	return nil
}

func resourceArmAvailabilitySetDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ArmClient).availSetClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	resGroup := id.ResourceGroup
	name := id.Path["availabilitySets"]

	_, err = client.Delete(ctx, resGroup, name)

	return diag.FromErr(err)
}
