package azurestack

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2016-04-01/dns"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurestack/azurestack/helpers/utils"
)

func resourceArmDnsARecord() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceArmDnsARecordCreateOrUpdate,
		ReadContext:   resourceArmDnsARecordRead,
		UpdateContext: resourceArmDnsARecordCreateOrUpdate,
		DeleteContext: resourceArmDnsARecordDelete,
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

			"zone_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"records": {
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"ttl": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmDnsARecordCreateOrUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	dnsClient := meta.(*ArmClient).dnsClient

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)
	zoneName := d.Get("zone_name").(string)
	ttl := int64(d.Get("ttl").(int))
	tags := d.Get("tags").(map[string]interface{})

	records, err := expandAzureStackDnsARecords(d)
	if err != nil {
		return diag.FromErr(err)
	}

	parameters := dns.RecordSet{
		Name: &name,
		RecordSetProperties: &dns.RecordSetProperties{
			Metadata: *expandTags(tags),
			TTL:      &ttl,
			ARecords: &records,
		},
	}

	eTag := ""
	ifNoneMatch := "" // set to empty to allow updates to records after creation
	resp, err := dnsClient.CreateOrUpdate(ctx, resGroup, zoneName, name, "A", parameters, eTag, ifNoneMatch)
	if err != nil {
		return diag.FromErr(err)
	}

	if resp.ID == nil {
		return diag.Errorf("Cannot read DNS A Record %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*resp.ID)

	return resourceArmDnsARecordRead(ctx, d, meta)
}

func resourceArmDnsARecordRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	dnsClient := meta.(*ArmClient).dnsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resGroup := id.ResourceGroup
	name := id.Path["A"]
	zoneName := id.Path["dnszones"]

	resp, err := dnsClient.Get(ctx, resGroup, zoneName, name, dns.A)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("Error reading DNS A record %s: %+v", name, err)
	}

	d.Set("name", name)
	d.Set("resource_group_name", resGroup)
	d.Set("zone_name", zoneName)
	d.Set("ttl", resp.TTL)

	if err := d.Set("records", flattenAzureStackDnsARecords(resp.ARecords)); err != nil {
		return diag.FromErr(err)
	}
	flattenAndSetTags(d, &resp.Metadata)

	return nil
}

func resourceArmDnsARecordDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	dnsClient := meta.(*ArmClient).dnsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resGroup := id.ResourceGroup
	name := id.Path["A"]
	zoneName := id.Path["dnszones"]

	resp, error := dnsClient.Delete(ctx, resGroup, zoneName, name, dns.A, "")
	if resp.StatusCode != http.StatusOK {
		return diag.Errorf("Error deleting DNS A Record %s: %+v", name, error)
	}

	return nil
}

func flattenAzureStackDnsARecords(records *[]dns.ARecord) []string {
	results := make([]string, 0, len(*records))

	if records != nil {
		for _, record := range *records {
			results = append(results, *record.Ipv4Address)
		}
	}

	return results
}

func expandAzureStackDnsARecords(d *schema.ResourceData) ([]dns.ARecord, error) {
	recordStrings := d.Get("records").(*schema.Set).List()
	records := make([]dns.ARecord, len(recordStrings))

	for i, v := range recordStrings {
		ipv4 := v.(string)
		records[i] = dns.ARecord{
			Ipv4Address: &ipv4,
		}
	}

	return records, nil
}
