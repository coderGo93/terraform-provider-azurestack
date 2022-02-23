package subscription_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-provider-azurestack/internal/clients"
	"github.com/hashicorp/terraform-provider-azurestack/internal/services/subscription/parse"
	"github.com/hashicorp/terraform-provider-azurestack/internal/tf/acceptance"
	"github.com/hashicorp/terraform-provider-azurestack/internal/tf/acceptance/check"
	"github.com/hashicorp/terraform-provider-azurestack/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurestack/internal/utils"
)

type SubscriptionResource struct{}

func TestAccSubscriptionResource_basic(t *testing.T) {

	data := acceptance.BuildTestData(t, "azurestack_subscription", "test")
	r := SubscriptionResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r)),
		},
		data.ImportStep(),
	})
}

func TestAccSubscriptionResource_requiresImport(t *testing.T) {
	if os.Getenv("ARM_BILLING_ACCOUNT") == "" {
		t.Skip("skipping tests - no billing account data provided")
	}

	data := acceptance.BuildTestData(t, "azurestack_subscription", "test")
	r := SubscriptionResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r)),
		},
		data.RequiresImportErrorStep(r.requiresImport),
	})
}

func TestAccSubscriptionResource_update(t *testing.T) {
	if os.Getenv("ARM_BILLING_ACCOUNT") == "" {
		t.Skip("skipping tests - no billing account data provided")
	}
	data := acceptance.BuildTestData(t, "azurestack_subscription", "test")
	r := SubscriptionResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r)),
		},
		data.ImportStep("billing_scope_id"),
		{
			Config: r.basicEnrollmentAccountUpdate(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r)),
		},
		data.ImportStep("billing_scope_id"),
	})
}

func TestAccSubscriptionResource_devTest(t *testing.T) {
	if os.Getenv("ARM_BILLING_ACCOUNT") == "" {
		t.Skip("skipping tests - no billing account data provided")
	}

	data := acceptance.BuildTestData(t, "azurestack_subscription", "test")
	r := SubscriptionResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.basicEnrollmentAccountDevTest(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r)),
		},
		data.ImportStep(),
	})
}

func (SubscriptionResource) Exists(ctx context.Context, client *clients.Client, state *pluginsdk.InstanceState) (*bool, error) {
	id, err := parse.SubscriptionAliasID(state.ID)
	if err != nil {
		return nil, err
	}
	resp, err := client.Subscription.AliasClient.Get(ctx, id.Name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			return utils.Bool(false), nil
		}
		return nil, fmt.Errorf("retrieving Subscription Alias %q: %+v", id.Name, err)
	}

	return utils.Bool(true), nil
}

func (SubscriptionResource) basic(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurestack" {
  features {}
}

resource "azurestack_subscription" "test" {
  alias             = "testAcc-%[1]d"
  subscription_name = "testAccSubscription %[1]d"
}
`, data.RandomInteger)
}

func (SubscriptionResource) basicEnrollmentAccountUpdate(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurestack" {
  features {}
}

resource "azurestack_subscription" "test" {
  alias             = "testAcc-%[1]d"
  subscription_name = "testAccSubscription Renamed %[1]d"
}
`, data.RandomInteger)
}

func (SubscriptionResource) basicEnrollmentAccountDevTest(data acceptance.TestData) string {
	billingAccount := os.Getenv("ARM_BILLING_ACCOUNT")
	enrollmentAccount := os.Getenv("ARM_BILLING_ENROLLMENT_ACCOUNT")
	return fmt.Sprintf(`
provider "azurestack" {
  features {}
}

data "azurestack_billing_enrollment_account_scope" "test" {
  billing_account_name    = "%s"
  enrollment_account_name = "%s"
}

resource "azurestack_subscription" "test" {
  alias             = "testAcc-%[3]d"
  subscription_name = "testAccSubscription Renamed %[3]d"
  billing_scope_id  = data.azurestack_billing_enrollment_account_scope.test.id
  workload          = "DevTest"
}
`, billingAccount, enrollmentAccount, data.RandomInteger)
}

func (r SubscriptionResource) requiresImport(data acceptance.TestData) string {
	return fmt.Sprintf(`
%s

resource "azurestack_subscription" "import" {
  alias             = azurestack_subscription.test.alias
  subscription_name = azurestack_subscription.test.subscription_name
  billing_scope_id  = azurestack_subscription.test.billing_scope_id
}
`, r.basic(data))
}
