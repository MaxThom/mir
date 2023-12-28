package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/protoproxy"
	"github.com/maxthom/mir/api/gen/proto/v1alpha/protoproxy/protoproxyconnect"

	"connectrpc.com/connect"
	logger "github.com/rs/zerolog/log"
)

func main() {
	client := protoproxyconnect.NewProtoProxyServiceClient(
		http.DefaultClient,
		"http://localhost:3000",
	)
	res, err := client.UploadSchema(
		context.Background(),
		connect.NewRequest(&protoproxy.UploadSchemaRequest{}),
	)
	if err != nil {
		logger.Error().Err(err)
		return
	}

	fmt.Println(fmt.Sprintf("%s", res))
	logger.Info().Msg(fmt.Sprintf("%s", res))
}
