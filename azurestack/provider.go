package azurestack

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"hash/crc32"
	"log"
	"strings"
	"sync"

	"github.com/hashicorp/go-azure-helpers/authentication"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			// TODO: deprecate this local key in favour of `endpoint` in the futures
			"arm_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_ENDPOINT", ""),
			},
			"subscription_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_SUBSCRIPTION_ID", ""),
			},

			"client_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_ID", ""),
			},

			"client_certificate_path": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_CERTIFICATE_PATH", ""),
			},

			"client_certificate_password": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_CERTIFICATE_PASSWORD", ""),
			},

			"client_secret": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_SECRET", ""),
			},

			"tenant_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_TENANT_ID", ""),
			},

			"skip_credentials_validation": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_SKIP_CREDENTIALS_VALIDATION", false),
			},

			"skip_provider_registration": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_SKIP_PROVIDER_REGISTRATION", false),
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"azurestack_client_config":           dataSourceArmClientConfig(),
			"azurestack_network_interface":       dataSourceArmNetworkInterface(),
			"azurestack_network_security_group":  dataSourceArmNetworkSecurityGroup(),
			"azurestack_platform_image":          dataSourceArmPlatformImage(),
			"azurestack_public_ip":               dataSourceArmPublicIP(),
			"azurestack_resource_group":          dataSourceArmResourceGroup(),
			"azurestack_storage_account":         dataSourceArmStorageAccount(),
			"azurestack_virtual_network":         dataSourceArmVirtualNetwork(),
			"azurestack_route_table":             dataSourceArmRouteTable(),
			"azurestack_subnet":                  dataSourceArmSubnet(),
			"azurestack_virtual_network_gateway": dataSourceArmVirtualNetworkGateway(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"azurestack_availability_set":                   resourceArmAvailabilitySet(),
			"azurestack_dns_zone":                           resourceArmDnsZone(),
			"azurestack_dns_a_record":                       resourceArmDnsARecord(),
			"azurestack_network_interface":                  resourceArmNetworkInterface(),
			"azurestack_network_security_group":             resourceArmNetworkSecurityGroup(),
			"azurestack_network_security_rule":              resourceArmNetworkSecurityRule(),
			"azurestack_local_network_gateway":              resourceArmLocalNetworkGateway(),
			"azurestack_lb":                                 resourceArmLoadBalancer(),
			"azurestack_lb_backend_address_pool":            resourceArmLoadBalancerBackendAddressPool(),
			"azurestack_lb_nat_rule":                        resourceArmLoadBalancerNatRule(),
			"azurestack_lb_probe":                           resourceArmLoadBalancerProbe(),
			"azurestack_lb_nat_pool":                        resourceArmLoadBalancerNatPool(),
			"azurestack_lb_rule":                            resourceArmLoadBalancerRule(),
			"azurestack_managed_disk":                       resourceArmManagedDisk(),
			"azurestack_public_ip":                          resourceArmPublicIp(),
			"azurestack_resource_group":                     resourceArmResourceGroup(),
			"azurestack_route":                              resourceArmRoute(),
			"azurestack_route_table":                        resourceArmRouteTable(),
			"azurestack_storage_account":                    resourceArmStorageAccount(),
			"azurestack_storage_blob":                       resourceArmStorageBlob(),
			"azurestack_storage_container":                  resourceArmStorageContainer(),
			"azurestack_subnet":                             resourceArmSubnet(),
			"azurestack_template_deployment":                resourceArmTemplateDeployment(),
			"azurestack_virtual_network":                    resourceArmVirtualNetwork(),
			"azurestack_virtual_network_gateway":            resourceArmVirtualNetworkGateway(),
			"azurestack_virtual_machine":                    resourceArmVirtualMachine(),
			"azurestack_virtual_machine_extension":          resourceArmVirtualMachineExtensions(),
			"azurestack_virtual_network_gateway_connection": resourceArmVirtualNetworkGatewayConnection(),
			"azurestack_virtual_machine_scale_set":          resourceArmVirtualMachineScaleSet(),
		},
	}

	p.ConfigureContextFunc = providerConfigure(p)

	return p
}

func providerConfigure(p *schema.Provider) schema.ConfigureContextFunc {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		builder := authentication.Builder{
			SubscriptionID:                d.Get("subscription_id").(string),
			ClientID:                      d.Get("client_id").(string),
			ClientSecret:                  d.Get("client_secret").(string),
			TenantID:                      d.Get("tenant_id").(string),
			ClientCertPath:                d.Get("client_certificate_path").(string),
			ClientCertPassword:            d.Get("client_certificate_password").(string),
			CustomResourceManagerEndpoint: d.Get("arm_endpoint").(string),
			Environment:                   "AZURESTACKCLOUD",

			// Feature Toggles
			SupportsAzureCliToken:    true,
			SupportsClientSecretAuth: true,
			SupportsClientCertAuth:   true,
		}
		config, err := builder.Build()
		if err != nil {
			return nil, diag.Errorf("Error building ARM Client: %+v", err)
		}

		skipCredentialsValidation := d.Get("skip_credentials_validation").(bool)
		skipProviderRegistration := d.Get("skip_provider_registration").(bool)
		client, err := getArmClient(config, p.TerraformVersion, skipProviderRegistration)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		//lint:ignore SA1019 SDKv2 migration - staticcheck's own linter directives are currently being ignored under golanci-lint
		stopCtx, ok := schema.StopContext(ctx) //nolint:staticcheck
		if !ok {
			stopCtx = ctx
		}

		client.StopContext = stopCtx

		if !skipCredentialsValidation {
			// List all the available providers and their registration state to avoid unnecessary
			// requests. This also lets us check if the provider credentials are correct.
			ctx := client.StopContext
			providerList, err := client.providersClient.List(ctx, nil, "")
			if err != nil {
				return nil, diag.Errorf("Unable to list provider registration status, it is possible that this is due to invalid "+
					"credentials or the service principal does not have permission to use the Resource Manager API, Azure "+
					"error: %s", err)
			}

			if !skipProviderRegistration {
				err = ensureResourceProvidersAreRegistered(ctx, client.providersClient, providerList.Values(), requiredResourceProviders())
				if err != nil {
					return nil, diag.FromErr(err)
				}
			}
		}

		return client, nil
	}
}

// armMutexKV is the instance of MutexKV for ARM resources
var armMutexKV = NewMutexKV()

// Resource group names can be capitalised, but we store them in lowercase.
// Use a custom diff function to avoid creation of new resources.
func resourceAzureStackResourceGroupNameDiffSuppress(_, old, new string, _ *schema.ResourceData) bool {
	return strings.EqualFold(old, new)
}

// ignoreCaseDiffSuppressFunc is a DiffSuppressFunc from helper/schema that is
// used to ignore any case-changes in a return value.
func ignoreCaseDiffSuppressFunc(_, old, new string, _ *schema.ResourceData) bool {
	return strings.EqualFold(old, new)
}

// ignoreCaseStateFunc is a StateFunc from helper/schema that converts the
// supplied value to lower before saving to state for consistency.
func ignoreCaseStateFunc(val interface{}) string {
	return strings.ToLower(val.(string))
}

func userDataStateFunc(v interface{}) string {
	switch s := v.(type) {
	case string:
		s = base64Encode(s)
		hash := sha1.Sum([]byte(s))
		return hex.EncodeToString(hash[:])
	default:
		return ""
	}
}

// base64Encode encodes data if the input isn't already encoded using
// base64.StdEncoding.EncodeToString. If the input is already base64 encoded,
// return the original input unchanged.
func base64Encode(data string) string {
	// Check whether the data is already Base64 encoded; don't double-encode
	if isBase64Encoded(data) {
		return data
	}
	// data has not been encoded encode and return
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func isBase64Encoded(data string) bool {
	_, err := base64.StdEncoding.DecodeString(data)
	return err == nil
}

func userDataDiffSuppressFunc(k, old, new string, d *schema.ResourceData) bool {
	oldValue := userDataStateFunc(old)
	return oldValue == new
}

// MutexKV is a simple key/value store for arbitrary mutexes. It can be used to
// serialize changes across arbitrary collaborators that share knowledge of the
// keys they must serialize on.
//
// The initial use case is to let aws_security_group_rule resources serialize
// their access to individual security groups based on SG ID.
type MutexKV struct {
	lock  sync.Mutex
	store map[string]*sync.Mutex
}

// Locks the mutex for the given key. Caller is responsible for calling Unlock
// for the same key
func (m *MutexKV) Lock(key string) {
	log.Printf("[DEBUG] Locking %q", key)
	m.get(key).Lock()
	log.Printf("[DEBUG] Locked %q", key)
}

// Unlock the mutex for the given key. Caller must have called Lock for the same key first
func (m *MutexKV) Unlock(key string) {
	log.Printf("[DEBUG] Unlocking %q", key)
	m.get(key).Unlock()
	log.Printf("[DEBUG] Unlocked %q", key)
}

// Returns a mutex for the given key, no guarantee of its lock status
func (m *MutexKV) get(key string) *sync.Mutex {
	m.lock.Lock()
	defer m.lock.Unlock()
	mutex, ok := m.store[key]
	if !ok {
		mutex = &sync.Mutex{}
		m.store[key] = mutex
	}
	return mutex
}

// Returns a properly initialized MutexKV
func NewMutexKV() *MutexKV {
	return &MutexKV{
		store: make(map[string]*sync.Mutex),
	}
}

// HashCodeString hashes a string to a unique hashcode.
//
// crc32 returns a uint32, but for our use we need
// and non negative integer. Here we cast to an integer
// and invert it if the result is negative.
func HashCodeString(s string) int {
	v := int(crc32.ChecksumIEEE([]byte(s)))
	if v >= 0 {
		return v
	}
	if -v >= 0 {
		return -v
	}
	// v == MinInt
	return 0
}
