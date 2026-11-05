package contabo

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// regionSlugs maps the region names the API returns for some resources (for
// example object storages) to the region slugs used in requests.
var regionSlugs = map[string]string{
	"european union":          "EU",
	"united states (central)": "US-central",
	"united states (east)":    "US-east",
	"united states (west)":    "US-west",
	"asia (singapore)":        "SIN",
	"united kingdom":          "UK",
	"australia (sydney)":      "AUS",
	"asia (japan)":            "JPN",
	"asia (india)":            "IND",
}

// normalizeRegion returns the region slug for a region slug or name.
func normalizeRegion(region string) string {
	trimmed := strings.TrimSpace(region)
	if slug, ok := regionSlugs[strings.ToLower(trimmed)]; ok {
		return slug
	}
	for _, slug := range regionSlugs {
		if strings.EqualFold(slug, trimmed) {
			return slug
		}
	}
	return trimmed
}

func suppressEquivalentRegion(_, old, new string, _ *schema.ResourceData) bool {
	return normalizeRegion(old) == normalizeRegion(new)
}
