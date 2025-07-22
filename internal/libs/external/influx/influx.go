package influx

import (
	"context"
	"fmt"
	"strings"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

type Options struct {
	BatchSize        uint
	FlushInterval    uint
	RetryBufferLimit uint
	UseGZip          bool
}

func DefaultOptions() Options {
	return Options{
		BatchSize:        1000,               // Number of datapoints
		FlushInterval:    1000,               // 1sec
		RetryBufferLimit: 1000 * 1024 * 1024, // 1GB
		UseGZip:          false,
	}
}

func NewConnectedClient(ctx context.Context, url string, token string) (influxdb2.Client, error) {
	lpClient := influxdb2.NewClient(url, token)
	if b, err := lpClient.Ping(ctx); !b || err != nil {
		return lpClient, err
	}
	return lpClient, nil
}

// NewConnectedClientWithOptions creates a new InfluxDB client with enhanced configuration
// for better retry policies and larger buffer sizes
func NewConnectedClientWithOptions(ctx context.Context, url string, token string, opts Options) (influxdb2.Client, error) {
	// Create client with custom options
	lpClient := influxdb2.NewClientWithOptions(url, token,
		influxdb2.DefaultOptions().
			SetBatchSize(opts.BatchSize).               // Default 1000
			SetFlushInterval(opts.FlushInterval).       // Flush every 1 seconds (default is 1s)
			SetRetryBufferLimit(opts.RetryBufferLimit). // 10GB retry buffer, default 50MB
			SetUseGZip(opts.UseGZip).                   // Enable gzip compression
			SetMaxRetries(100000).                      // More retries
			SetMaxRetryTime(60*60*24*7*1000).           // Max 7 days of total retry time
			SetRetryInterval(5000).                     // 5 second base retry interval
			SetMaxRetryInterval(30*1000).               // Max 30 seconds retry interval
			SetExponentialBase(2).                      // Exponential backoff
			SetHTTPRequestTimeout(60))                  // 60 second timeout (default is 20s)

	// Test connection
	if b, err := lpClient.Ping(ctx); !b || err != nil {
		return lpClient, err
	}

	return lpClient, nil
}

func CreateOrgAndBucket(ctx context.Context, lpClient influxdb2.Client, orgName, bucketName string) error {
	orgApi := lpClient.OrganizationsAPI()
	org, err := orgApi.FindOrganizationByName(ctx, orgName)
	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("not found: organization name \"%s\" not found", orgName)) {
			org, err = orgApi.CreateOrganizationWithName(ctx, orgName)
			if err != nil {
				return err
			}
		} else {
			return err
		}
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
