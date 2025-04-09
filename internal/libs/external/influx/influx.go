package influx

import (
	"context"
	"fmt"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
	"github.com/rs/zerolog"
)

func NewConnectedClient(ctx context.Context, l zerolog.Logger, url string, token string) influxdb2.Client {
	lpClient := influxdb2.NewClient(url, token)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		if b, _ := lpClient.Ping(ctx); b {
			break
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			l.Warn().Msg("InfluxDB connection failed, attempting to reconnect...")
		}
	}

	return lpClient
}

func CreateOrgAndBucket(ctx context.Context, lpClient influxdb2.Client, orgName, bucketName string) error {
	orgApi := lpClient.OrganizationsAPI()
	org, err := orgApi.FindOrganizationByName(ctx, orgName)
	if err != nil {
		// TODO check if we can create org
		// if strings.Contains(err.Error(), fmt.Sprintf("not found: organization name \"%s\" not found", orgName)) {
		// 	org, err = orgApi.CreateOrganizationWithName(ctx, orgName)
		// 	if err != nil {
		// 		return err
		// 	}
		// }
		return err
	}

	_, err = lpClient.BucketsAPI().CreateBucketWithName(ctx, org, bucketName, domain.RetentionRule{
		EverySeconds: 0,
	})
	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("conflict: bucket with name %s already exists", bucketName)) {
			return nil
		}
		return err
	}
	return nil
}
