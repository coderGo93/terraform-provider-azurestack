package storage

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tombuildsstuff/giovanni/storage/2018-11-09/blob/blobs"

	"github.com/hashicorp/terraform-provider-azurestack/internal/clients"
	"github.com/hashicorp/terraform-provider-azurestack/internal/services/storage/migration"
	"github.com/hashicorp/terraform-provider-azurestack/internal/services/storage/validate"
	"github.com/hashicorp/terraform-provider-azurestack/internal/tf"
	"github.com/hashicorp/terraform-provider-azurestack/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurestack/internal/tf/timeouts"
	"github.com/hashicorp/terraform-provider-azurestack/internal/utils"
)

func storageBlob() *schema.Resource {
	return &schema.Resource{
		Create:        storageBlobCreate,
		Read:          storageBlobRead,
		Delete:        storageBlobDelete,
		Update:        storageBlobUpdate,
		SchemaVersion: 1,
		StateUpgraders: pluginsdk.StateUpgrades(map[int]pluginsdk.StateUpgrade{
			0: migration.BlobV0ToV1{},
		}),

		// TODO: replace this with an importer which validates the ID during import
		Importer: pluginsdk.DefaultImporter(),

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(120 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(120 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(120 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// TODO: add validation
			},

			"storage_account_name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.StorageAccountName,
			},

			"storage_container_name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.StorageContainerName,
			},

			"type": {
				Type:     pluginsdk.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"Append",
					"Block",
					"Page",
				}, false),
			},

			"size": {
				Type:         pluginsdk.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Default:      0,
				ValidateFunc: validation.IntDivisibleBy(512),
			},

			"content_type": {
				Type:     pluginsdk.TypeString,
				Optional: true,
				Default:  "application/octet-stream",
			},

			"cache_control": {
				Type:     pluginsdk.TypeString,
				Optional: true,
			},

			"source": {
				Type:          pluginsdk.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"source_uri", "source_content"},
			},

			"source_content": {
				Type:          pluginsdk.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"source", "source_uri"},
			},

			"source_uri": {
				Type:          pluginsdk.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"source", "source_content"},
			},

			"content_md5": {
				Type:          pluginsdk.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"source_uri"},
			},

			"url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"parallelism": {
				// TODO: @tombuildsstuff - a note this only works for Page blobs
				Type:         pluginsdk.TypeInt,
				Optional:     true,
				Default:      8,
				ForceNew:     true,
				ValidateFunc: validation.IntAtLeast(1),
			},

			"metadata": MetaDataComputedSchema(),
		},
	}
}

func storageBlobCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	storageClient := meta.(*clients.Client).Storage
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	accountName := d.Get("storage_account_name").(string)
	containerName := d.Get("storage_container_name").(string)
	name := d.Get("name").(string)

	account, err := storageClient.FindAccount(ctx, accountName)
	if err != nil {
		return fmt.Errorf("retrieving Account %q for Blob %q (Container %q): %s", accountName, name, containerName, err)
	}
	if account == nil {
		return fmt.Errorf("Unable to locate Storage Account %q!", accountName)
	}

	blobsClient, err := storageClient.BlobsClient(ctx, *account)
	if err != nil {
		return fmt.Errorf("building Blobs Client: %s", err)
	}

	id := blobsClient.GetResourceID(accountName, containerName, name)
	if d.IsNewResource() {
		input := blobs.GetPropertiesInput{}
		props, err := blobsClient.GetProperties(ctx, accountName, containerName, name, input)
		if err != nil {
			if !utils.ResponseWasNotFound(props.Response) {
				return fmt.Errorf("checking if Blob %q exists (Container %q / Account %q / Resource Group %q): %s", name, containerName, accountName, account.ResourceGroup, err)
			}
		}
		if !utils.ResponseWasNotFound(props.Response) {
			return tf.ImportAsExistsError("azurestack_storage_blob", id)
		}
	}

	contentMD5Raw := d.Get("content_md5").(string)
	contentMD5 := ""
	if contentMD5Raw != "" {
		// Azure uses a Base64 encoded representation of the standard MD5 sum of the file
		contentMD5, err = convertHexToBase64Encoding(d.Get("content_md5").(string))
		if err != nil {
			return fmt.Errorf("failed to base64 encode `content_md5` value: %s", err)
		}
	}

	log.Printf("[DEBUG] Creating Blob %q in Container %q within Storage Account %q..", name, containerName, accountName)
	metaDataRaw := d.Get("metadata").(map[string]interface{})
	blobInput := BlobUpload{
		AccountName:   accountName,
		ContainerName: containerName,
		BlobName:      name,
		Client:        blobsClient,

		BlobType:      d.Get("type").(string),
		CacheControl:  d.Get("cache_control").(string),
		ContentType:   d.Get("content_type").(string),
		ContentMD5:    contentMD5,
		MetaData:      ExpandMetaData(metaDataRaw),
		Parallelism:   d.Get("parallelism").(int),
		Size:          d.Get("size").(int),
		Source:        d.Get("source").(string),
		SourceContent: d.Get("source_content").(string),
		SourceUri:     d.Get("source_uri").(string),
	}
	if err := blobInput.Create(ctx); err != nil {
		return fmt.Errorf("creating Blob %q (Container %q / Account %q): %s", name, containerName, accountName, err)
	}
	log.Printf("[DEBUG] Created Blob %q in Container %q within Storage Account %q.", name, containerName, accountName)

	d.SetId(id)

	return storageBlobUpdate(d, meta)
}

func storageBlobUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	storageClient := meta.(*clients.Client).Storage
	ctx, cancel := timeouts.ForUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := blobs.ParseResourceID(d.Id())
	if err != nil {
		return fmt.Errorf("parsing %q: %s", d.Id(), err)
	}

	account, err := storageClient.FindAccount(ctx, id.AccountName)
	if err != nil {
		return fmt.Errorf("retrieving Account %q for Blob %q (Container %q): %s", id.AccountName, id.BlobName, id.ContainerName, err)
	}
	if account == nil {
		return fmt.Errorf("Unable to locate Storage Account %q!", id.AccountName)
	}

	blobsClient, err := storageClient.BlobsClient(ctx, *account)
	if err != nil {
		return fmt.Errorf("building Blobs Client: %s", err)
	}

	if d.HasChange("content_type") || d.HasChange("cache_control") {
		log.Printf("[DEBUG] Updating Properties for Blob %q (Container %q / Account %q)...", id.BlobName, id.ContainerName, id.AccountName)
		contentType := d.Get("content_type").(string)
		cacheControl := d.Get("cache_control").(string)
		input := blobs.SetPropertiesInput{
			ContentType:  &contentType,
			CacheControl: &cacheControl,
		}

		// `content_md5` is `ForceNew` but must be included in the `SetPropertiesInput` update payload or it will be zeroed on the blob.
		if contentMD5 := d.Get("content_md5").(string); contentMD5 != "" {
			data, err := convertHexToBase64Encoding(contentMD5)
			if err != nil {
				return fmt.Errorf("in converting hex to base64 encoding for content_md5: %s", err)
			}

			contentMD5data := data
			input.ContentMD5 = &contentMD5data
		}

		if _, err := blobsClient.SetProperties(ctx, id.AccountName, id.ContainerName, id.BlobName, input); err != nil {
			return fmt.Errorf("updating Properties for Blob %q (Container %q / Account %q): %s", id.BlobName, id.ContainerName, id.AccountName, err)
		}
		log.Printf("[DEBUG] Updated Properties for Blob %q (Container %q / Account %q).", id.BlobName, id.ContainerName, id.AccountName)
	}

	if d.HasChange("metadata") {
		log.Printf("[DEBUG] Updating MetaData for Blob %q (Container %q / Account %q)...", id.BlobName, id.ContainerName, id.AccountName)
		metaDataRaw := d.Get("metadata").(map[string]interface{})
		input := blobs.SetMetaDataInput{
			MetaData: ExpandMetaData(metaDataRaw),
		}
		if _, err := blobsClient.SetMetaData(ctx, id.AccountName, id.ContainerName, id.BlobName, input); err != nil {
			return fmt.Errorf("updating MetaData for Blob %q (Container %q / Account %q): %s", id.BlobName, id.ContainerName, id.AccountName, err)
		}
		log.Printf("[DEBUG] Updated MetaData for Blob %q (Container %q / Account %q).", id.BlobName, id.ContainerName, id.AccountName)
	}

	return storageBlobRead(d, meta)
}

func storageBlobRead(d *pluginsdk.ResourceData, meta interface{}) error {
	storageClient := meta.(*clients.Client).Storage
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := blobs.ParseResourceID(d.Id())
	if err != nil {
		return fmt.Errorf("parsing %q: %s", d.Id(), err)
	}

	account, err := storageClient.FindAccount(ctx, id.AccountName)
	if err != nil {
		return fmt.Errorf("retrieving Account %q for Blob %q (Container %q): %s", id.AccountName, id.BlobName, id.ContainerName, err)
	}
	if account == nil {
		log.Printf("[DEBUG] Unable to locate Account %q for Blob %q (Container %q) - assuming removed & removing from state!", id.AccountName, id.BlobName, id.ContainerName)
		d.SetId("")
		return nil
	}

	blobsClient, err := storageClient.BlobsClient(ctx, *account)
	if err != nil {
		return fmt.Errorf("building Blobs Client: %s", err)
	}

	log.Printf("[INFO] Retrieving Storage Blob %q (Container %q / Account %q).", id.BlobName, id.ContainerName, id.AccountName)
	input := blobs.GetPropertiesInput{}
	props, err := blobsClient.GetProperties(ctx, id.AccountName, id.ContainerName, id.BlobName, input)
	if err != nil {
		if utils.ResponseWasNotFound(props.Response) {
			log.Printf("[INFO] Blob %q was not found in Container %q / Account %q - assuming removed & removing from state...", id.BlobName, id.ContainerName, id.AccountName)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("retrieving properties for Blob %q (Container %q / Account %q): %s", id.BlobName, id.ContainerName, id.AccountName, err)
	}

	d.Set("name", id.BlobName)
	d.Set("storage_container_name", id.ContainerName)
	d.Set("storage_account_name", id.AccountName)

	d.Set("content_type", props.ContentType)
	d.Set("cache_control", props.CacheControl)

	// Set the ContentMD5 value to md5 hash in hex
	contentMD5 := ""
	if props.ContentMD5 != "" {
		contentMD5, err = convertBase64ToHexEncoding(props.ContentMD5)
		if err != nil {
			return fmt.Errorf("in converting hex to base64 encoding for content_md5: %s", err)
		}
	}
	d.Set("content_md5", contentMD5)

	d.Set("type", strings.TrimSuffix(string(props.BlobType), "Blob"))
	d.Set("url", d.Id())

	if err := d.Set("metadata", FlattenMetaData(props.MetaData)); err != nil {
		return fmt.Errorf("setting `metadata`: %+v", err)
	}

	// The CopySource is only returned if the blob hasn't been modified (e.g. metadata configured etc)
	// as such, we need to conditionally set this to ensure it's trackable if possible
	if props.CopySource != "" {
		d.Set("source_uri", props.CopySource)
	}

	return nil
}

func storageBlobDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	storageClient := meta.(*clients.Client).Storage
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := blobs.ParseResourceID(d.Id())
	if err != nil {
		return fmt.Errorf("parsing %q: %s", d.Id(), err)
	}

	account, err := storageClient.FindAccount(ctx, id.AccountName)
	if err != nil {
		return fmt.Errorf("retrieving Account %q for Blob %q (Container %q): %s", id.AccountName, id.BlobName, id.ContainerName, err)
	}
	if account == nil {
		return fmt.Errorf("Unable to locate Storage Account %q!", id.AccountName)
	}

	blobsClient, err := storageClient.BlobsClient(ctx, *account)
	if err != nil {
		return fmt.Errorf("building Blobs Client: %s", err)
	}

	log.Printf("[INFO] Deleting Blob %q from Container %q / Storage Account %q", id.BlobName, id.ContainerName, id.AccountName)
	input := blobs.DeleteInput{
		DeleteSnapshots: true,
	}
	if _, err := blobsClient.Delete(ctx, id.AccountName, id.ContainerName, id.BlobName, input); err != nil {
		return fmt.Errorf("deleting Blob %q (Container %q / Account %q): %s", id.BlobName, id.ContainerName, id.AccountName, err)
	}

	return nil
}
