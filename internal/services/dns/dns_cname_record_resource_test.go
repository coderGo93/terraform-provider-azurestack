package dns_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/dns/mgmt/dns"
	"github.com/hashicorp/terraform-provider-azurestack/internal/clients"
	"github.com/hashicorp/terraform-provider-azurestack/internal/services/dns/parse"
	"github.com/hashicorp/terraform-provider-azurestack/internal/tf/acceptance"
	"github.com/hashicorp/terraform-provider-azurestack/internal/tf/acceptance/check"
	"github.com/hashicorp/terraform-provider-azurestack/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurestack/internal/utils"
)

type DnsCNameRecordResource struct{}

func TestAccDnsCNameRecord_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurestack_dns_cname_record", "test")
	r := DnsCNameRecordResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("fqdn").Exists(),
			),
		},
		data.ImportStep(),
	})
}

func TestAccDnsCNameRecord_requiresImport(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurestack_dns_cname_record", "test")
	r := DnsCNameRecordResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		{
			Config:      r.requiresImport(data),
			ExpectError: acceptance.RequiresImportError("azurestack_dns_cname_record"),
		},
	})
}

func TestAccDnsCNameRecord_subdomain(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurestack_dns_cname_record", "test")
	r := DnsCNameRecordResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.subdomain(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("record").HasValue("test.contoso.com"),
			),
		},
		data.ImportStep(),
	})
}

func TestAccDnsCNameRecord_updateRecords(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurestack_dns_cname_record", "test")
	r := DnsCNameRecordResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		{
			Config: r.updateRecords(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
	})
}

func TestAccDnsCNameRecord_withTags(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurestack_dns_cname_record", "test")
	r := DnsCNameRecordResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.withTags(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("tags.%").HasValue("2"),
			),
		},
		{
			Config: r.withTagsUpdate(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("tags.%").HasValue("1"),
			),
		},
		data.ImportStep(),
	})
}

func (DnsCNameRecordResource) Exists(ctx context.Context, clients *clients.Client, state *pluginsdk.InstanceState) (*bool, error) {
	id, err := parse.CnameRecordID(state.ID)
	if err != nil {
		return nil, err
	}

	resp, err := clients.Dns.RecordSetsClient.Get(ctx, id.ResourceGroup, id.DnszoneName, id.CNAMEName, dns.CNAME)
	if err != nil {
		return nil, fmt.Errorf("retrieving DNS CNAME record %s (resource group: %s): %v", id.CNAMEName, id.ResourceGroup, err)
	}

	return utils.Bool(resp.RecordSetProperties != nil), nil
}

func (DnsCNameRecordResource) basic(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurestack" {
  features {}
}

resource "azurestack_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurestack_dns_zone" "test" {
  name                = "acctestzone%d.com"
  resource_group_name = azurestack_resource_group.test.name
}

resource "azurestack_dns_cname_record" "test" {
  name                = "myarecord%d"
  resource_group_name = azurestack_resource_group.test.name
  zone_name           = azurestack_dns_zone.test.name
  ttl                 = 300
  record              = "contoso.com"
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger)
}

func (r DnsCNameRecordResource) requiresImport(data acceptance.TestData) string {
	return fmt.Sprintf(`
%s

resource "azurestack_dns_cname_record" "import" {
  name                = azurestack_dns_cname_record.test.name
  resource_group_name = azurestack_dns_cname_record.test.resource_group_name
  zone_name           = azurestack_dns_cname_record.test.zone_name
  ttl                 = 300
  record              = "contoso.com"
}
`, r.basic(data))
}

func (DnsCNameRecordResource) subdomain(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurestack" {
  features {}
}

resource "azurestack_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurestack_dns_zone" "test" {
  name                = "acctestzone%d.com"
  resource_group_name = azurestack_resource_group.test.name
}

resource "azurestack_dns_cname_record" "test" {
  name                = "myarecord%d"
  resource_group_name = azurestack_resource_group.test.name
  zone_name           = azurestack_dns_zone.test.name
  ttl                 = 300
  record              = "test.contoso.com"
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger)
}

func (DnsCNameRecordResource) updateRecords(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurestack" {
  features {}
}

resource "azurestack_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurestack_dns_zone" "test" {
  name                = "acctestzone%d.com"
  resource_group_name = azurestack_resource_group.test.name
}

resource "azurestack_dns_cname_record" "test" {
  name                = "myarecord%d"
  resource_group_name = azurestack_resource_group.test.name
  zone_name           = azurestack_dns_zone.test.name
  ttl                 = 300
  record              = "contoso.co.uk"
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger)
}

func (DnsCNameRecordResource) withTags(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurestack" {
  features {}
}

resource "azurestack_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurestack_dns_zone" "test" {
  name                = "acctestzone%d.com"
  resource_group_name = azurestack_resource_group.test.name
}

resource "azurestack_dns_cname_record" "test" {
  name                = "myarecord%d"
  resource_group_name = azurestack_resource_group.test.name
  zone_name           = azurestack_dns_zone.test.name
  ttl                 = 300
  record              = "contoso.com"

  tags = {
    environment = "Production"
    cost_center = "MSFT"
  }
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger)
}

func (DnsCNameRecordResource) withTagsUpdate(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurestack" {
  features {}
}

resource "azurestack_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurestack_dns_zone" "test" {
  name                = "acctestzone%d.com"
  resource_group_name = azurestack_resource_group.test.name
}

resource "azurestack_dns_cname_record" "test" {
  name                = "myarecord%d"
  resource_group_name = azurestack_resource_group.test.name
  zone_name           = azurestack_dns_zone.test.name
  ttl                 = 300
  record              = "contoso.com"

  tags = {
    environment = "staging"
  }
}
`, data.RandomInteger, data.Locations.Primary, data.RandomInteger, data.RandomInteger)
}
