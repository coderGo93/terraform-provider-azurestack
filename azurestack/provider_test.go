package azurestack

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/go-azure-helpers/authentication"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider
var testAccProvidersFactories map[string]func() (*schema.Provider, error)

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"azurestack": testAccProvider,
	}
	testAccProvidersFactories = map[string]func() (*schema.Provider, error){
		"azurestack": func() (*schema.Provider, error) { return testAccProvider, nil },
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ = Provider()
}

func testAccPreCheck(t *testing.T) {
	variables := []string{
		"ARM_CLIENT_ID",
		"ARM_CLIENT_SECRET",
		"ARM_SUBSCRIPTION_ID",
		"ARM_TENANT_ID",
		"ARM_TEST_LOCATION",
	}

	for _, variable := range variables {
		value := os.Getenv(variable)
		if value == "" {
			t.Fatalf("`%s` must be set for acceptance tests!", variable)
		}
	}
}

func testLocation() string {
	return os.Getenv("ARM_TEST_LOCATION")
}

func testAltLocation() string {
	return os.Getenv("ARM_TEST_LOCATION_ALT")
}

func testGetAzureConfig(t *testing.T) *authentication.Config {
	if os.Getenv(resource.TestEnvVar) == "" {
		t.Skip(fmt.Sprintf("Integration test skipped unless env '%s' set", resource.TestEnvVar))
		return nil
	}

	// we deliberately don't use the main config - since we care about
	builder := authentication.Builder{
		SubscriptionID:                os.Getenv("ARM_SUBSCRIPTION_ID"),
		ClientID:                      os.Getenv("ARM_CLIENT_ID"),
		TenantID:                      os.Getenv("ARM_TENANT_ID"),
		ClientSecret:                  os.Getenv("ARM_CLIENT_SECRET"),
		CustomResourceManagerEndpoint: os.Getenv("ARM_ENDPOINT"),
		Environment:                   "AZURESTACKCLOUD",

		// Feature Toggles
		SupportsClientSecretAuth: true,
	}

	config, err := builder.Build()
	if err != nil {
		t.Fatalf("Error building ARM Client: %+v", err)
		return nil
	}

	return config
}
