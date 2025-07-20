package influx

import (
	"context"
	"fmt"
	"strings"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

func NewConnectedClient(ctx context.Context, url string, token string) (influxdb2.Client, error) {
	lpClient := influxdb2.NewClient(url, token)
	if b, err := lpClient.Ping(ctx); !b || err != nil {
		return lpClient, err
	}
	return lpClient, nil
}

func CreateOrgAndBucket(ctx context.Context, lpClient influxdb2.Client, orgName, bucketName string) error {
	orgApi := lpClient.OrganizationsAPI()
	org, err := orgApi.FindOrganizationByName(ctx, orgName)
	if err != nil {
		// TODO check if we can create org
		if strings.Contains(err.Error(), fmt.Sprintf("not found: organization name \"%s\" not found", orgName)) {
			org, err = orgApi.CreateOrganizationWithName(ctx, orgName)
			if err != nil {
				return err
			}
		}
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
