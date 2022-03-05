package compute_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-provider-azurestack/internal/tf/acceptance"
	"github.com/hashicorp/terraform-provider-azurestack/internal/tf/acceptance/check"
)

func TestAccWindowsVirtualMachineScaleSet_disksOSDiskCaching(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurestack_windows_virtual_machine_scale_set", "test")
	r := WindowsVirtualMachineScaleSetResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.disksOSDiskCaching(data, "None"),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep("admin_password"),
		{
			Config: r.disksOSDiskCaching(data, "ReadOnly"),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep("admin_password"),
		{
			Config: r.disksOSDiskCaching(data, "ReadWrite"),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep("admin_password"),
	})
}

func TestAccWindowsVirtualMachineScaleSet_disksOSDiskCustomSize(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurestack_windows_virtual_machine_scale_set", "test")
	r := WindowsVirtualMachineScaleSetResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			// unset
			Config: r.authPassword(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep("admin_password"),
		{
			Config: r.disksOSDiskCustomSize(data, 128),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep("admin_password"),
		{
			// resize a second time to confirm https://github.com/Azure/azure-rest-api-specs/issues/1906
			Config: r.disksOSDiskCustomSize(data, 256),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep("admin_password"),
	})
}

func TestAccWindowsVirtualMachineScaleSet_disksOSDiskEphemeral(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurestack_windows_virtual_machine_scale_set", "test")
	r := WindowsVirtualMachineScaleSetResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.disksOSDiskEphemeral(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep("admin_password"),
	})
}

func TestAccWindowsVirtualMachineScaleSet_disksOSDiskDiskEncryptionSet(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurestack_windows_virtual_machine_scale_set", "test")
	r := WindowsVirtualMachineScaleSetResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.disksOSDisk_diskEncryptionSet(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep("admin_password"),
	})
}

func TestAccWindowsVirtualMachineScaleSet_disksOSDiskStorageAccountTypeStandardLRS(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurestack_windows_virtual_machine_scale_set", "test")
	r := WindowsVirtualMachineScaleSetResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.disksOSDiskStorageAccountType(data, "Standard_LRS"),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep("admin_password"),
	})
}

func TestAccWindowsVirtualMachineScaleSet_disksOSDiskStorageAccountTypeStandardSSDLRS(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurestack_windows_virtual_machine_scale_set", "test")
	r := WindowsVirtualMachineScaleSetResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.disksOSDiskStorageAccountType(data, "StandardSSD_LRS"),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep("admin_password"),
	})
}

func TestAccWindowsVirtualMachineScaleSet_disksOSDiskStorageAccountTypePremiumLRS(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurestack_windows_virtual_machine_scale_set", "test")
	r := WindowsVirtualMachineScaleSetResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.disksOSDiskStorageAccountType(data, "Premium_LRS"),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep("admin_password"),
	})
}

func TestAccWindowsVirtualMachineScaleSet_disksOSDiskWriteAcceleratorEnabled(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurestack_windows_virtual_machine_scale_set", "test")
	r := WindowsVirtualMachineScaleSetResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.disksOSDiskWriteAcceleratorEnabled(data, true),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep("admin_password"),
	})
}

func (r WindowsVirtualMachineScaleSetResource) disksOSDiskCaching(data acceptance.TestData, caching string) string {
	return fmt.Sprintf(`
%s

resource "azurestack_windows_virtual_machine_scale_set" "test" {
  name                = local.vm_name
  resource_group_name = azurestack_resource_group.test.name
  location            = azurestack_resource_group.test.location
  sku                 = "Standard_F2"
  instances           = 1
  admin_username      = "adminuser"
  admin_password      = "P@ssword1234!"

  source_image_reference {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2019-Datacenter"
    version   = "latest"
  }

  os_disk {
    storage_account_type = "Standard_LRS"
    caching              = "%s"
  }

  network_interface {
    name    = "example"
    primary = true

    ip_configuration {
      name      = "internal"
      primary   = true
      subnet_id = azurestack_subnet.test.id
    }
  }
}
`, r.template(data), caching)
}

func (r WindowsVirtualMachineScaleSetResource) disksOSDiskCustomSize(data acceptance.TestData, diskSize int) string {
	return fmt.Sprintf(`
%s

resource "azurestack_windows_virtual_machine_scale_set" "test" {
  name                = local.vm_name
  resource_group_name = azurestack_resource_group.test.name
  location            = azurestack_resource_group.test.location
  sku                 = "Standard_F2"
  instances           = 1
  admin_username      = "adminuser"
  admin_password      = "P@ssword1234!"

  source_image_reference {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2019-Datacenter"
    version   = "latest"
  }

  os_disk {
    storage_account_type = "Standard_LRS"
    caching              = "ReadWrite"
    disk_size_gb         = %d
  }

  network_interface {
    name    = "example"
    primary = true

    ip_configuration {
      name      = "internal"
      primary   = true
      subnet_id = azurestack_subnet.test.id
    }
  }
}
`, r.template(data), diskSize)
}

func (WindowsVirtualMachineScaleSetResource) disksOSDisk_diskEncryptionSetDependencies(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurestack" {
  features {
    key_vault {
      recover_soft_deleted_key_vaults = false
      purge_soft_delete_on_destroy    = false
    }
  }
}

locals {
  vm_name = "accVM-%d"
}
data "azurestack_client_config" "current" {}

resource "azurestack_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"
}

resource "azurestack_key_vault" "test" {
  name                        = "acctestkv%s"
  location                    = azurestack_resource_group.test.location
  resource_group_name         = azurestack_resource_group.test.name
  tenant_id                   = data.azurestack_client_config.current.tenant_id
  sku_name                    = "standard"
  purge_protection_enabled    = true
  enabled_for_disk_encryption = true
}

resource "azurestack_key_vault_access_policy" "service-principal" {
  key_vault_id = azurestack_key_vault.test.id
  tenant_id    = data.azurestack_client_config.current.tenant_id
  object_id    = data.azurestack_client_config.current.object_id

  key_permissions = [
    "Create",
    "Delete",
    "Get",
    "Purge",
    "Update",
  ]

  secret_permissions = [
    "Get",
    "Delete",
    "Set",
  ]
}

resource "azurestack_key_vault_key" "test" {
  name         = "examplekey"
  key_vault_id = azurestack_key_vault.test.id
  key_type     = "RSA"
  key_size     = 2048

  key_opts = [
    "Decrypt",
    "Encrypt",
    "Sign",
    "UnwrapKey",
    "Verify",
    "WrapKey",
  ]

  depends_on = ["azurestack_key_vault_access_policy.service-principal"]
}

resource "azurestack_virtual_network" "test" {
  name                = "acctestnw-%d"
  address_space       = ["10.0.0.0/16"]
  location            = azurestack_resource_group.test.location
  resource_group_name = azurestack_resource_group.test.name
}

resource "azurestack_subnet" "test" {
  name                 = "internal"
  resource_group_name  = azurestack_resource_group.test.name
  virtual_network_name = azurestack_virtual_network.test.name
  address_prefix       = "10.0.2.0/24"
}
`, data.RandomInteger, data.RandomInteger, data.Locations.Primary, data.RandomString, data.RandomInteger)
}

func (r WindowsVirtualMachineScaleSetResource) disksOSDisk_diskEncryptionSetResource(data acceptance.TestData) string {
	return fmt.Sprintf(`
%s

resource "azurestack_disk_encryption_set" "test" {
  name                = "acctestdes-%d"
  resource_group_name = azurestack_resource_group.test.name
  location            = azurestack_resource_group.test.location
  key_vault_key_id    = azurestack_key_vault_key.test.id

  identity {
    type = "SystemAssigned"
  }
}

resource "azurestack_key_vault_access_policy" "disk-encryption" {
  key_vault_id = azurestack_key_vault.test.id

  key_permissions = [
    "Get",
    "WrapKey",
    "UnwrapKey",
  ]

  tenant_id = azurestack_disk_encryption_set.test.identity.0.tenant_id
  object_id = azurestack_disk_encryption_set.test.identity.0.principal_id
}

resource "azurestack_role_assignment" "disk-encryption-read-keyvault" {
  scope                = azurestack_key_vault.test.id
  role_definition_name = "Reader"
  principal_id         = azurestack_disk_encryption_set.test.identity.0.principal_id
}
`, r.disksOSDisk_diskEncryptionSetDependencies(data), data.RandomInteger)
}

func (r WindowsVirtualMachineScaleSetResource) disksOSDisk_diskEncryptionSet(data acceptance.TestData) string {
	return fmt.Sprintf(`
%s

resource "azurestack_windows_virtual_machine_scale_set" "test" {
  name                 = local.vm_name
  resource_group_name  = azurestack_resource_group.test.name
  location             = azurestack_resource_group.test.location
  sku                  = "Standard_F2"
  instances            = 1
  admin_username       = "adminuser"
  admin_password       = "P@ssword1234!"
  computer_name_prefix = "destest"

  source_image_reference {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2019-Datacenter"
    version   = "latest"
  }

  os_disk {
    storage_account_type   = "Standard_LRS"
    caching                = "ReadOnly"
    disk_encryption_set_id = azurestack_disk_encryption_set.test.id
  }

  network_interface {
    name    = "example"
    primary = true

    ip_configuration {
      name      = "internal"
      primary   = true
      subnet_id = azurestack_subnet.test.id
    }
  }

  depends_on = [
    "azurestack_role_assignment.disk-encryption-read-keyvault",
    "azurestack_key_vault_access_policy.disk-encryption",
  ]
}
`, r.disksOSDisk_diskEncryptionSetResource(data))
}

func (r WindowsVirtualMachineScaleSetResource) disksOSDiskEphemeral(data acceptance.TestData) string {
	return fmt.Sprintf(`
%s

resource "azurestack_windows_virtual_machine_scale_set" "test" {
  name                = local.vm_name
  resource_group_name = azurestack_resource_group.test.name
  location            = azurestack_resource_group.test.location
  sku                 = "Standard_F8s_v2" # has to be this large for ephemeral disks on Windows
  instances           = 1
  admin_username      = "adminuser"
  admin_password      = "P@ssword1234!"

  source_image_reference {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2019-Datacenter"
    version   = "latest"
  }

  os_disk {
    storage_account_type = "Standard_LRS"
    caching              = "ReadOnly"

    diff_disk_settings {
      option = "Local"
    }
  }

  network_interface {
    name    = "example"
    primary = true

    ip_configuration {
      name      = "internal"
      primary   = true
      subnet_id = azurestack_subnet.test.id
    }
  }
}
`, r.template(data))
}

func (r WindowsVirtualMachineScaleSetResource) disksOSDiskStorageAccountType(data acceptance.TestData, storageAccountType string) string {
	return fmt.Sprintf(`
%s

resource "azurestack_windows_virtual_machine_scale_set" "test" {
  name                = local.vm_name
  resource_group_name = azurestack_resource_group.test.name
  location            = azurestack_resource_group.test.location
  sku                 = "Standard_F2s_v2"
  instances           = 1
  admin_username      = "adminuser"
  admin_password      = "P@ssword1234!"

  source_image_reference {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2019-Datacenter"
    version   = "latest"
  }

  os_disk {
    storage_account_type = %q
    caching              = "ReadWrite"
  }

  network_interface {
    name    = "example"
    primary = true

    ip_configuration {
      name      = "internal"
      primary   = true
      subnet_id = azurestack_subnet.test.id
    }
  }
}
`, r.template(data), storageAccountType)
}

func (r WindowsVirtualMachineScaleSetResource) disksOSDiskWriteAcceleratorEnabled(data acceptance.TestData, enabled bool) string {
	return fmt.Sprintf(`
%s

resource "azurestack_windows_virtual_machine_scale_set" "test" {
  name                = local.vm_name
  resource_group_name = azurestack_resource_group.test.name
  location            = azurestack_resource_group.test.location
  sku                 = "Standard_M8ms"
  instances           = 1
  admin_username      = "adminuser"
  admin_password      = "P@ssword1234!"

  source_image_reference {
    publisher = "MicrosoftWindowsServer"
    offer     = "WindowsServer"
    sku       = "2019-Datacenter"
    version   = "latest"
  }

  os_disk {
    storage_account_type      = "Premium_LRS"
    caching                   = "None"
    write_accelerator_enabled = %t
  }

  network_interface {
    name    = "example"
    primary = true

    ip_configuration {
      name      = "internal"
      primary   = true
      subnet_id = azurestack_subnet.test.id
    }
  }
}
`, r.template(data), enabled)
}
