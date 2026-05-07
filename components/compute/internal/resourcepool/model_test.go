package resourcepool

import "testing"

func TestResourcePool_TableName(t *testing.T) {
	if (ResourcePool{}).TableName() != "resource_pools" {
		t.Error("table name mismatch")
	}
}
