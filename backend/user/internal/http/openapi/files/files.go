// Package files serves locally stored uploads through signed URLs when the
// storage provider is "local". The signature is the credential.
package files

import (
	"net/http"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/storage"
)

var handler http.Handler

// Configure installs the local storage handler; a nil provider disables the route.
func Configure(p storage.Provider) {
	if local, ok := p.(*storage.Local); ok {
		handler = local.Handler()
	}
}

// GetRouteSpecs returns the files route (empty when storage is remote).
func GetRouteSpecs() []gohttp.RouteSpec {
	if handler == nil {
		return nil
	}
	return []gohttp.RouteSpec{{Method: "GET", Path: "/openapi/files/*", Handler: handler.ServeHTTP, TimeOut: 120}}
}
