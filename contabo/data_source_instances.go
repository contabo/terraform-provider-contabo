package contabo

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"contabo.com/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceInstances() *schema.Resource {
	instanceAttributes := instanceComputedAttributes()
	instanceAttributes["id"] = &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The id of the instance.",
	}
	instanceAttributes["tag_ids"] = &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Elem:        &schema.Schema{Type: schema.TypeInt},
		Description: "The ids of the tags from the `tag_ids` filter. The list is empty when you do not use the `tag_ids` filter.",
	}

	return &schema.Resource{
		Description: "This data source lists compute instances. You can filter the list, and an instance must match every filter that you set. Use this data source to find existing instances, for example before you import the instances with `terraform import`.",
		ReadContext: dataSourceInstancesRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "When you set this filter, the list only contains the instance with this name, for example `vmi123456`.",
			},
			"display_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "When you set this filter, the list only contains instances with exactly this display name.",
			},
			"display_name_prefix": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "When you set this filter, the list only contains instances whose display name starts with this text.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "When you set this filter, the list only contains instances in this region, for example `EU`.",
			},
			"product_ids": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "When you set this filter, the list only contains instances of these products, for example `V153`.",
			},
			"product_types": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "When you set this filter, the list only contains instances of these product types, for example `ssd` or `nvme`.",
			},
			"statuses": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "When you set this filter, the list only contains instances with one of these statuses, for example `running`.",
			},
			"tag_ids": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Description: "When you set this filter, the list only contains instances that have every one of these tags.",
			},
			"include_cancelled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the list contains instances that have a cancellation date. The default value is `true`.",
			},
			"ids": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The ids of the instances that match the filters, sorted from the lowest id to the highest id.",
			},
			"instances": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The instances that match the filters, sorted by id.",
				Elem:        &schema.Resource{Schema: instanceAttributes},
			},
		},
	}
}

func dataSourceInstancesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*openapi.APIClient)

	query := url.Values{}
	if name := d.Get("name").(string); name != "" {
		query.Set("name", name)
	}
	if displayName := d.Get("display_name").(string); displayName != "" {
		query.Set("displayName", displayName)
	}
	if region := d.Get("region").(string); region != "" {
		query.Set("region", region)
	}
	productIds := stringSetToSlice(d.Get("product_ids").(*schema.Set))
	productTypes := stringSetToSlice(d.Get("product_types").(*schema.Set))
	statuses := stringSetToSlice(d.Get("statuses").(*schema.Set))

	instances, httpResp, err := listInstances(ctx, client, query)
	if err != nil {
		return apiErrorDiags(httpResp, err)
	}

	tagIds := setToInt64s(d.Get("tag_ids").(*schema.Set))
	var taggedIds map[string]bool
	if len(tagIds) > 0 {
		taggedIds, err = instancesWithAllTags(ctx, client, tagIds)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	filter := instanceFilter{
		name:             d.Get("name").(string),
		displayName:      d.Get("display_name").(string),
		displayPrefix:    d.Get("display_name_prefix").(string),
		region:           d.Get("region").(string),
		productIds:       productIds,
		productTypes:     productTypes,
		statuses:         statuses,
		includeCancelled: d.Get("include_cancelled").(bool),
		taggedIds:        taggedIds,
	}

	var matches []instanceResponse
	for _, instance := range instances {
		if filter.matches(instance) {
			matches = append(matches, instance)
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].InstanceId < matches[j].InstanceId })

	ids := make([]string, 0, len(matches))
	flattened := make([]interface{}, 0, len(matches))
	for _, instance := range matches {
		id := strconv.FormatInt(instance.InstanceId, 10)
		ids = append(ids, id)
		values := flattenInstance(instance)
		values["id"] = id
		delete(values, "additional_ips_v4")
		if taggedIds != nil {
			values["tag_ids"] = int64sToInterfaces(tagIds)
		} else {
			values["tag_ids"] = []interface{}{}
		}
		for key := range values {
			if _, ok := dataSourceInstances().Schema["instances"].Elem.(*schema.Resource).Schema[key]; !ok {
				delete(values, key)
			}
		}
		flattened = append(flattened, values)
	}

	d.SetId(fmt.Sprintf("instances-%d", hashStrings(append(ids, filter.String()))))
	if err := d.Set("ids", ids); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("instances", flattened); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

type instanceFilter struct {
	name             string
	displayName      string
	displayPrefix    string
	region           string
	productIds       []string
	productTypes     []string
	statuses         []string
	includeCancelled bool
	taggedIds        map[string]bool
}

func (f instanceFilter) matches(instance instanceResponse) bool {
	if f.name != "" && instance.Name != f.name {
		return false
	}
	if f.displayName != "" && instance.DisplayName != f.displayName {
		return false
	}
	if f.displayPrefix != "" && !strings.HasPrefix(instance.DisplayName, f.displayPrefix) {
		return false
	}
	if f.region != "" && !strings.EqualFold(instance.Region, f.region) {
		return false
	}
	if len(f.productIds) > 0 && !containsFold(f.productIds, instance.ProductId) {
		return false
	}
	if len(f.productTypes) > 0 && !containsFold(f.productTypes, instance.ProductType) {
		return false
	}
	if len(f.statuses) > 0 && !containsFold(f.statuses, instance.Status) {
		return false
	}
	if !f.includeCancelled && instance.cancelDate() != "" {
		return false
	}
	if f.taggedIds != nil && !f.taggedIds[strconv.FormatInt(instance.InstanceId, 10)] {
		return false
	}
	return true
}

func (f instanceFilter) String() string {
	return fmt.Sprintf("%s|%s|%s|%s|%v|%v|%v|%v", f.name, f.displayName, f.displayPrefix, f.region, f.productIds, f.productTypes, f.statuses, f.includeCancelled)
}

func containsFold(values []string, value string) bool {
	for _, v := range values {
		if strings.EqualFold(v, value) {
			return true
		}
	}
	return false
}

// instancesWithAllTags returns the ids of the instances that carry every tag.
func instancesWithAllTags(ctx context.Context, client *openapi.APIClient, tagIds []int64) (map[string]bool, error) {
	var result map[string]bool
	for _, tagId := range tagIds {
		tagged := map[string]bool{}
		path := fmt.Sprintf("/v1/tags/%d/assignments", tagId)
		_, err := listAll(ctx, client, path, url.Values{"resourceType": []string{"instance"}}, func(body []byte) (int64, error) {
			var page struct {
				Data []struct {
					ResourceType string `json:"resourceType"`
					ResourceId   string `json:"resourceId"`
				} `json:"data"`
				Pagination pagination `json:"_pagination"`
			}
			if err := jsonUnmarshal(body, &page); err != nil {
				return 0, err
			}
			for _, assignment := range page.Data {
				if strings.EqualFold(assignment.ResourceType, "instance") {
					tagged[assignment.ResourceId] = true
				}
			}
			return page.Pagination.TotalPages, nil
		})
		if err != nil {
			return nil, fmt.Errorf("listing assignments of tag %d: %w", tagId, err)
		}
		if result == nil {
			result = tagged
			continue
		}
		for id := range result {
			if !tagged[id] {
				delete(result, id)
			}
		}
	}
	return result, nil
}
