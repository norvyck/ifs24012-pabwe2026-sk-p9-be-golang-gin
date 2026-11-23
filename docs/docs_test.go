package docs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSwaggerJSON(t *testing.T) {
	defaultJSON := GetSwaggerJSON("", "")
	assert.Contains(t, defaultJSON, `"host": "localhost:8081"`)
	assert.Contains(t, defaultJSON, `"schemes": ["http"]`)

	customMap := ParseSwaggerMap("api.delcom.org", "https")
	assert.Equal(t, "api.delcom.org", customMap["host"])
	assert.Equal(t, "2.0", customMap["swagger"])
	assert.NotNil(t, customMap["paths"])
}
