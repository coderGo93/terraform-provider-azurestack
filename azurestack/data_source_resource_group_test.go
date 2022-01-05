package azurestack

import (
	"testing"

	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceAzureStackResourceGroup_basic(t *testing.T) {
	ri := acctest.RandInt()
	name := fmt.Sprintf("acctestRg_%d", ri)
	location := testLocation()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAzureStackResourceGroupBasic(name, location),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.azurestack_resource_group.test", "name", name),
					resource.TestCheckResourceAttr("data.azurestack_resource_group.test", "location", azureStackNormalizeLocation(location)),
					resource.TestCheckResourceAttr("data.azurestack_resource_group.test", "tags.%", "1"),
					resource.TestCheckResourceAttr("data.azurestack_resource_group.test", "tags.env", "test"),
				),
			},
		},
	})
}

func testAccDataSourceAzureStackResourceGroupBasic(name string, location string) string {
	return fmt.Sprintf(`
resource "azurestack_resource_group" "test" {
  name     = "%s"
  location = "%s"

  tags = {
    env = "test"
  }
}

data "azurestack_resource_group" "test" {
  name = "${azurestack_resource_group.test.name}"
}
`, name, location)
}
