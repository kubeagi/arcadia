// inspire by https://github.com/tmc/langchaingo/pull/702
package langchainwrap

import (
	"net/http"
	"net/http/httputil"

	"k8s.io/klog/v2"
)

// DebugHTTPClient is a http.Client that logs the request and response with full contents.
var DebugHTTPClient = &http.Client{ //nolint:gochecknoglobals
	Transport: &logTransport{http.DefaultTransport},
}

type logTransport struct {
	Transport http.RoundTripper
}

// RoundTrip logs the request and response with full contents using httputil.DumpRequest and httputil.DumpResponse.
func (t *logTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if klog.FromContext(req.Context()).V(5).Enabled() {
		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			return nil, err
		}
		klog.FromContext(req.Context()).V(5).Info(string(dump))
	}
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if klog.FromContext(req.Context()).V(5).Enabled() {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return nil, err
		}
		klog.FromContext(req.Context()).V(5).Info(string(dump))
	}
	return resp, nil
}
