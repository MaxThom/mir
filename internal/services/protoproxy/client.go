package protoproxy

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/maxthom/mir/api/gen/proto/v1alpha/protoproxy/protoproxyconnect"
)

var ()

func NewProtoProxyClient(httpClient *http.Client, baseUrl string, opts ...connect.ClientOption) protoproxyconnect.ProtoProxyServiceClient {
	return protoproxyconnect.NewProtoProxyServiceClient(httpClient, baseUrl, opts...)
}
