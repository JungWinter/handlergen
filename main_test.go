package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToSnakeCase(t *testing.T) {
	cases := []struct {
		given    string
		expected string
	}{
		{"", ""},
		{"already_snake", "already_snake"},
		{"A", "a"},
		{"AA", "aa"},
		{"AaAa", "aa_aa"},
		{"SignUp", "sign_up"},
		{"JungWinter", "jung_winter"},
		{"HTTPRequest", "http_request"},
		{"OurAPI", "our_api"},
	}
	for _, tc := range cases {
		actual := toSnakeCase(tc.given)

		assert.Equal(t, tc.expected, actual)
	}
}

func TestParseProto(t *testing.T) {
	f, err := os.Open("testdata/test.proto")
	assert.NoError(t, err)

	t.Run("test proto", func(t *testing.T) {
		p, err := parseProto(f)

		assert.NoError(t, err)
		assert.Equal(t, proto{
			GoPackage:   "github.com/myorg/myproto/sample",
			ServiceName: "sample",
			RPCNames: []string{
				"SignIn",
				"SignUp",
			},
		}, p)
	})
	t.Run("test grpc gateway proto with comment", func(t *testing.T) {
		p := `syntax = "proto3";

package v1.sample;

option go_package = "github.com/myorg/myproto/sample";

service Sample {
  // no whitespace between rpc and request message
  rpc SignIn(SignInRequest) returns (SignInResponse) {
    option (google.api.http) = {
      post: "/v1/sample/sign-in"
    };
  }
  // whitespace between rpc and request message
  rpc SignUp (SignUpRequest) returns (SignUpResponse) {
    option (google.api.http) = {
      post: "/v1/sample/sign-up"
    };
  }
}
`

		actual, err := parseProto(strings.NewReader(p))

		assert.NoError(t, err)
		assert.Equal(t, proto{
			GoPackage:   "github.com/myorg/myproto/sample",
			ServiceName: "sample",
			RPCNames: []string{
				"SignIn",
				"SignUp",
			},
		}, actual)
	})
	t.Run("no go package option", func(t *testing.T) {
		p := `syntax = "proto3";

package v1.sample;
`
		_, err := parseProto(strings.NewReader(p))
		assert.EqualError(t, err, "no go package option")
	})
	t.Run("no service name", func(t *testing.T) {
		p := `syntax = "proto3";

package v1.sample;

option go_package = "github.com/myorg/myproto/sample";
`
		_, err := parseProto(strings.NewReader(p))
		assert.EqualError(t, err, "no service name")

	})
	t.Run("no rpc", func(t *testing.T) {
		p := `syntax = "proto3";

package v1.sample;

option go_package = "github.com/myorg/myproto/sample";

service Sample {
}
`
		_, err := parseProto(strings.NewReader(p))
		assert.EqualError(t, err, "no rpcs")

	})
}
func TestWrite(t *testing.T) {
	t.Run("custom template", func(t *testing.T) {
		h := handler{
			GoPackage:   "github.com/myorg/myproto/sample",
			ServiceName: "sample",
			RPCName:     "SignUp",
		}
		tmplStr := "{{.GoPackage}}\n{{.ServiceName}}\n{{.RPCName}}"
		var tpl bytes.Buffer

		err := write(&tpl, tmplStr, h)

		assert.NoError(t, err)
		assert.Equal(t, "github.com/myorg/myproto/sample\nsample\nSignUp", tpl.String())
	})
	t.Run("handler template", func(t *testing.T) {
		h := handler{
			GoPackage:   "github.com/myorg/myproto/sample",
			ServiceName: "sample",
			RPCName:     "SignUp",
		}
		var tpl bytes.Buffer

		err := write(&tpl, handlerTmplStr, h)

		assert.NoError(t, err)

		f, err := os.Open("testdata/sign_up_handler.go.out")
		assert.NoError(t, err)
		var expected bytes.Buffer
		_, err = expected.ReadFrom(f)
		err = f.Close()
		assert.NoError(t, err)

		assert.Equal(t, expected.String(), tpl.String())
	})
	t.Run("handler test template", func(t *testing.T) {
		h := handler{
			GoPackage:   "github.com/myorg/myproto/sample",
			ServiceName: "sample",
			RPCName:     "SignUp",
		}
		var tpl bytes.Buffer

		err := write(&tpl, handlerTestTmplStr, h)

		assert.NoError(t, err)

		f, err := os.Open("testdata/sign_up_handler_test.go.out")
		assert.NoError(t, err)
		var expected bytes.Buffer
		_, err = expected.ReadFrom(f)
		err = f.Close()
		assert.NoError(t, err)

		assert.Equal(t, expected.String(), tpl.String())
	})
}
