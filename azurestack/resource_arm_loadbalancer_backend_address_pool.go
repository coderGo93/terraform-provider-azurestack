package azurestack

import (
	"context"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-10-01/network"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/terraform-providers/terraform-provider-azurestack/azurestack/helpers/utils"
)

func resourceArmLoadBalancerBackendAddressPool() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceArmLoadBalancerBackendAddressPoolCreate,
		ReadContext:   resourceArmLoadBalancerBackendAddressPoolRead,
		DeleteContext: resourceArmLoadBalancerBackendAddressPoolDelete,
		Importer: &schema.ResourceImporter{
			StateContext: loadBalancerSubResourceStateImporter,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": resourceGroupNameSchema(),

			"loadbalancer_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"backend_ip_configurations": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"load_balancing_rules": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceArmLoadBalancerBackendAddressPoolCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ArmClient).loadBalancerClient

	loadBalancerID := d.Get("loadbalancer_id").(string)
	armMutexKV.Lock(loadBalancerID)
	defer armMutexKV.Unlock(loadBalancerID)

	loadBalancer, exists, err := retrieveLoadBalancerById(ctx, loadBalancerID, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer %q not found. Removing from state", d.Get("name").(string))
		return nil
	}

	backendAddressPools := append(*loadBalancer.LoadBalancerPropertiesFormat.BackendAddressPools, expandAzureRmLoadBalancerBackendAddressPools(d))
	existingPool, existingPoolIndex, exists := findLoadBalancerBackEndAddressPoolByName(loadBalancer, d.Get("name").(string))
	if exists {
		if d.Get("name").(string) == *existingPool.Name {
			// this pool is being updated/reapplied remove old copy from the slice
			backendAddressPools = append(backendAddressPools[:existingPoolIndex], backendAddressPools[existingPoolIndex+1:]...)
		}
	}

	loadBalancer.LoadBalancerPropertiesFormat.BackendAddressPools = &backendAddressPools
	resGroup, loadBalancerName, err := resourceGroupAndLBNameFromId(d.Get("loadbalancer_id").(string))
	if err != nil {
		return diag.Errorf("Error parsing LoadBalancer Name and Group: %+v", err)
	}

	future, err := client.CreateOrUpdate(ctx, resGroup, loadBalancerName, *loadBalancer)
	if err != nil {
		return diag.Errorf("Error Creating/Updating LoadBalancer %q (Resource Group %q): %+v", loadBalancerName, resGroup, err)
	}

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err != nil {
		return diag.Errorf("Error Creating/Updating LoadBalancer %q (Resource Group %q): %+v", loadBalancerName, resGroup, err)
	}

	read, err := client.Get(ctx, resGroup, loadBalancerName, "")
	if err != nil {
		return diag.Errorf("Error retrieving Load Balancer %q (Resource Group %q): %+v", loadBalancerName, resGroup, err)
	}
	if read.ID == nil {
		return diag.Errorf("Cannot read LoadBalancer %q (Resource Group %q) ID", loadBalancerName, resGroup)
	}

	if read.LoadBalancerPropertiesFormat == nil {
		return diag.Errorf("Error creating LoadBalancer (%q Resource Group %q)", loadBalancerName, resGroup)
	}

	var poolId string
	for _, BackendAddressPool := range *(*read.LoadBalancerPropertiesFormat).BackendAddressPools {
		if *BackendAddressPool.Name == d.Get("name").(string) {
			poolId = *BackendAddressPool.ID
		}
	}

	if poolId == "" {
		return diag.Errorf("Cannot find created LoadBalancer Backend Address Pool ID %q", poolId)
	}

	d.SetId(poolId)

	// TODO: is this still needed?
	log.Printf("[DEBUG] Waiting for LoadBalancer (%s) to become available", loadBalancerName)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  []string{"Succeeded"},
		Refresh: loadbalancerStateRefreshFunc(ctx, client, resGroup, loadBalancerName),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return diag.Errorf("Error waiting for LoadBalancer (%q Resource Group %q) to become available: %+v", loadBalancerName, resGroup, err)
	}

	return resourceArmLoadBalancerBackendAddressPoolRead(ctx, d, meta)
}

func resourceArmLoadBalancerBackendAddressPoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	name := id.Path["backendAddressPools"]

	loadBalancer, exists, err := retrieveLoadBalancerById(ctx, d.Get("loadbalancer_id").(string), meta)
	if err != nil {
		return diag.Errorf("Error retrieving Load Balancer by ID: %+v", err)
	}
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer %q not found. Removing from state", name)
		return nil
	}

	config, _, exists := findLoadBalancerBackEndAddressPoolByName(loadBalancer, name)
	if !exists {
		d.SetId("")
		log.Printf("[INFO] LoadBalancer Backend Address Pool %q not found. Removing from state", name)
		return nil
	}

	d.Set("name", config.Name)
	d.Set("resource_group_name", id.ResourceGroup)

	var backendIpConfigurations []string
	var loadBalancingRules []string

	if props := config.BackendAddressPoolPropertiesFormat; props != nil {
		if configs := props.BackendIPConfigurations; configs != nil {
			for _, backendConfig := range *configs {
				backendIpConfigurations = append(backendIpConfigurations, *backendConfig.ID)
			}
		}

		if rules := props.LoadBalancingRules; rules != nil {
			for _, rule := range *rules {
				loadBalancingRules = append(loadBalancingRules, *rule.ID)
			}
		}
	}

	d.Set("backend_ip_configurations", backendIpConfigurations)
	d.Set("load_balancing_rules", loadBalancingRules)

	return nil
}

func resourceArmLoadBalancerBackendAddressPoolDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ArmClient).loadBalancerClient

	loadBalancerID := d.Get("loadbalancer_id").(string)
	armMutexKV.Lock(loadBalancerID)
	defer armMutexKV.Unlock(loadBalancerID)

	loadBalancer, exists, err := retrieveLoadBalancerById(ctx, loadBalancerID, meta)
	if err != nil {
		return diag.Errorf("Error retrieving Load Balancer by ID: %+v", err)
	}
	if !exists {
		d.SetId("")
		return nil
	}

	_, index, exists := findLoadBalancerBackEndAddressPoolByName(loadBalancer, d.Get("name").(string))
	if !exists {
		return nil
	}

	oldBackEndPools := *loadBalancer.LoadBalancerPropertiesFormat.BackendAddressPools
	newBackEndPools := append(oldBackEndPools[:index], oldBackEndPools[index+1:]...)
	loadBalancer.LoadBalancerPropertiesFormat.BackendAddressPools = &newBackEndPools

	resGroup, loadBalancerName, err := resourceGroupAndLBNameFromId(d.Get("loadbalancer_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	future, err := client.CreateOrUpdate(ctx, resGroup, loadBalancerName, *loadBalancer)
	if err != nil {
		return diag.Errorf("Error Creating/Updating LoadBalancer: %+v", err)
	}

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err != nil {
		return diag.Errorf("Error waiting for the completion for the LoadBalancer: %+v", err)
	}

	read, err := client.Get(ctx, resGroup, loadBalancerName, "")
	if err != nil {
		return diag.Errorf("Error retrieving the LoadBalancer %q (Resource Group %q): %+v", loadBalancerName, resGroup, err)
	}
	if read.ID == nil {
		return diag.Errorf("Cannot read LoadBalancer %q (resource group %q) ID", loadBalancerName, resGroup)
	}

	return nil
}

func expandAzureRmLoadBalancerBackendAddressPools(d *schema.ResourceData) network.BackendAddressPool {
	return network.BackendAddressPool{
		Name: utils.String(d.Get("name").(string)),
	}
}
