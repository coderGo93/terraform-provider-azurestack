package azurestack

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"regexp"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceArmStorageContainer() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceArmStorageContainerCreate,
		ReadContext:   resourceArmStorageContainerRead,
		DeleteContext: resourceArmStorageContainerDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArmStorageContainerName,
			},
			"resource_group_name": resourceGroupNameSchema(),
			"storage_account_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"container_access_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "private",
				ValidateFunc: validateArmStorageContainerAccessType,
			},
			"properties": {
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

//Following the naming convention as laid out in the docs
func validateArmStorageContainerName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^\$root$|^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in %q: %q",
			k, value))
	}
	if len(value) < 3 || len(value) > 63 {
		errors = append(errors, fmt.Errorf(
			"%q must be between 3 and 63 characters: %q", k, value))
	}
	if regexp.MustCompile(`^-`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot begin with a hyphen: %q", k, value))
	}
	return
}

func validateArmStorageContainerAccessType(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	validTypes := map[string]struct{}{
		"private":   {},
		"blob":      {},
		"container": {},
	}

	if _, ok := validTypes[value]; !ok {
		errors = append(errors, fmt.Errorf("Storage container access type %q is invalid, must be %q, %q or %q", value, "private", "blob", "page"))
	}
	return
}

func resourceArmStorageContainerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	blobClient, accountExists, err := armClient.getBlobStorageClientForStorageAccount(ctx, resourceGroupName, storageAccountName)
	if err != nil {
		return diag.FromErr(err)
	}
	if !accountExists {
		return diag.Errorf("Storage Account %q Not Found", storageAccountName)
	}

	name := d.Get("name").(string)

	var accessType storage.ContainerAccessType
	if d.Get("container_access_type").(string) == "private" {
		accessType = storage.ContainerAccessType("")
	} else {
		accessType = storage.ContainerAccessType(d.Get("container_access_type").(string))
	}

	log.Printf("[INFO] Creating container %q in storage account %q.", name, storageAccountName)
	reference := blobClient.GetContainerReference(name)

	err = resource.Retry(120*time.Second, checkContainerIsCreated(reference))
	if err != nil {
		return diag.Errorf("Error creating container %q in storage account %q: %s", name, storageAccountName, err)
	}

	permissions := storage.ContainerPermissions{
		AccessType: accessType,
	}
	permissionOptions := &storage.SetContainerPermissionOptions{}
	err = reference.SetPermissions(permissions, permissionOptions)
	if err != nil {
		return diag.Errorf("Error setting permissions for container %s in storage account %s: %+v", name, storageAccountName, err)
	}

	d.SetId(name)
	return resourceArmStorageContainerRead(ctx, d, meta)
}

func checkContainerIsCreated(reference *storage.Container) func() *resource.RetryError {
	return func() *resource.RetryError {
		createOptions := &storage.CreateContainerOptions{}
		_, err := reference.CreateIfNotExists(createOptions)
		if err != nil {
			return resource.RetryableError(err)
		}

		return nil
	}
}

// resourceAzureStorageContainerRead does all the necessary API calls to
// read the status of the storage container off Azure.
func resourceArmStorageContainerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	blobClient, accountExists, err := armClient.getBlobStorageClientForStorageAccount(ctx, resourceGroupName, storageAccountName)
	if err != nil {
		return diag.FromErr(err)
	}
	if !accountExists {
		log.Printf("[DEBUG] Storage account %q not found, removing container %q from state", storageAccountName, d.Id())
		d.SetId("")
		return nil
	}

	name := d.Get("name").(string)
	containers, err := blobClient.ListContainers(storage.ListContainersParameters{
		Prefix:  name,
		Timeout: 90,
	})
	if err != nil {
		return diag.Errorf("Failed to retrieve storage containers in account %q: %s", name, err)
	}

	var found bool
	for _, cont := range containers.Containers {
		if cont.Name == name {
			found = true

			props := make(map[string]interface{})
			props["last_modified"] = cont.Properties.LastModified
			props["lease_status"] = cont.Properties.LeaseStatus
			props["lease_state"] = cont.Properties.LeaseState
			props["lease_duration"] = cont.Properties.LeaseDuration

			d.Set("properties", props)
		}
	}

	if !found {
		log.Printf("[INFO] Storage container %q does not exist in account %q, removing from state...", name, storageAccountName)
		d.SetId("")
	}

	return nil
}

// resourceAzureStorageContainerDelete does all the necessary API calls to
// delete a storage container off Azure.
func resourceArmStorageContainerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	armClient := meta.(*ArmClient)

	resourceGroupName := d.Get("resource_group_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	blobClient, accountExists, err := armClient.getBlobStorageClientForStorageAccount(ctx, resourceGroupName, storageAccountName)
	if err != nil {
		return diag.FromErr(err)
	}
	if !accountExists {
		log.Printf("[INFO]Storage Account %q doesn't exist so the container won't exist", storageAccountName)
		return nil
	}

	name := d.Get("name").(string)

	log.Printf("[INFO] Deleting storage container %q in account %q", name, storageAccountName)
	reference := blobClient.GetContainerReference(name)
	deleteOptions := &storage.DeleteContainerOptions{}
	if _, err := reference.DeleteIfExists(deleteOptions); err != nil {
		return diag.Errorf("Error deleting storage container %q from storage account %q: %s", name, storageAccountName, err)
	}

	d.SetId("")
	return nil
}
