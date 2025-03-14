package broker

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHideSensitiveDataFromContext(t *testing.T) {
	// This test is just a way to show, how the hideSensitiveDataFromContext works
	in := map[string]interface{}{
		"password": "pa2345",
		"username": "johnsmith",
		"subobject": map[string]interface{}{
			"secret": "val",
			"sm_url": "https://sm.url.com",
		},
		"isValid": true,
	}

	// when
	out := hideSensitiveDataFromContext(in)

	d, err := json.Marshal(out)
	require.NoError(t, err)
	assert.Equal(t, `{"isValid":true,"password":"*****","subobject":{"secret":"*****","sm_url":"https://sm.url.com"},"username":"*****"}`, string(d))
}

func Test_hideSensitiveDataFromRawContextInputWithNullValue(t *testing.T) {
	// when
	out := hideSensitiveDataFromRawContext([]byte(`
{
    "xaappname": null
}
`))

	assert.Equal(t, map[string]interface{}{"xaappname": nil}, out)
}

func TestInputWithNullValue(t *testing.T) {
	// when
	out := hideSensitiveDataFromContext(map[string]interface{}{"xappname": nil})

	assert.Equal(t, map[string]interface{}{"xappname": nil}, out)
}
