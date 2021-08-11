// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/cortexproject/cortex/blob/master/integration/querier_streaming_mixed_ingester_test.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Cortex Authors.
// +build requires_docker

package integration

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/user"

	"github.com/grafana/mimir/integration/e2e"
	e2edb "github.com/grafana/mimir/integration/e2e/db"
	"github.com/grafana/mimir/integration/e2emimir"
	ingester_client "github.com/grafana/mimir/pkg/ingester/client"
	"github.com/grafana/mimir/pkg/mimirpb"
)

func TestQuerierWithStreamingBlocksAndChunksIngesters(t *testing.T) {
	for _, streamChunks := range []bool{false, true} {
		t.Run(fmt.Sprintf("%v", streamChunks), func(t *testing.T) {
			testQuerierWithStreamingBlocksAndChunksIngesters(t, streamChunks)
		})
	}
}

func testQuerierWithStreamingBlocksAndChunksIngesters(t *testing.T, streamChunks bool) {
	s, err := e2e.NewScenario(networkName)
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, writeFileToSharedDir(s, mimirSchemaConfigFile, []byte(mimirSchemaConfigYaml)))
	chunksFlags := ChunksStorageFlags()
	blockFlags := mergeFlags(BlocksStorageFlags(), map[string]string{
		"-blocks-storage.tsdb.block-ranges-period":      "1h",
		"-blocks-storage.tsdb.head-compaction-interval": "1m",
		"-store-gateway.sharding-enabled":               "false",
		"-querier.ingester-streaming":                   "true",
	})
	blockFlags["-ingester.stream-chunks-when-using-blocks"] = fmt.Sprintf("%v", streamChunks)

	// Start dependencies.
	consul := e2edb.NewConsul()
	minio := e2edb.NewMinio(9000, blockFlags["-blocks-storage.s3.bucket-name"])
	require.NoError(t, s.StartAndWaitReady(consul, minio))

	// Start Mimir components.
	ingesterBlocks := e2emimir.NewIngester("ingester-blocks", consul.NetworkHTTPEndpoint(), blockFlags, "")
	ingesterChunks := e2emimir.NewIngester("ingester-chunks", consul.NetworkHTTPEndpoint(), chunksFlags, "")
	storeGateway := e2emimir.NewStoreGateway("store-gateway", consul.NetworkHTTPEndpoint(), blockFlags, "")
	require.NoError(t, s.StartAndWaitReady(ingesterBlocks, ingesterChunks, storeGateway))

	// Sharding is disabled, pass gateway address.
	querierFlags := mergeFlags(blockFlags, map[string]string{
		"-querier.store-gateway-addresses": strings.Join([]string{storeGateway.NetworkGRPCEndpoint()}, ","),
		"-distributor.shard-by-all-labels": "true",
	})
	querier := e2emimir.NewQuerier("querier", consul.NetworkHTTPEndpoint(), querierFlags, "")
	require.NoError(t, s.StartAndWaitReady(querier))

	require.NoError(t, querier.WaitSumMetrics(e2e.Equals(1024), "cortex_ring_tokens_total"))

	s1 := []mimirpb.Sample{
		{Value: 1, TimestampMs: 1000},
		{Value: 2, TimestampMs: 2000},
		{Value: 3, TimestampMs: 3000},
		{Value: 4, TimestampMs: 4000},
		{Value: 5, TimestampMs: 5000},
	}

	s2 := []mimirpb.Sample{
		{Value: 1, TimestampMs: 1000},
		{Value: 2.5, TimestampMs: 2500},
		{Value: 3, TimestampMs: 3000},
		{Value: 5.5, TimestampMs: 5500},
	}

	clientConfig := ingester_client.Config{}
	clientConfig.RegisterFlags(flag.NewFlagSet("unused", flag.ContinueOnError)) // registers default values

	// Push data to chunks ingester.
	{
		ingesterChunksClient, err := ingester_client.MakeIngesterClient(ingesterChunks.GRPCEndpoint(), clientConfig)
		require.NoError(t, err)
		defer ingesterChunksClient.Close()

		_, err = ingesterChunksClient.Push(user.InjectOrgID(context.Background(), "user"), &mimirpb.WriteRequest{
			Timeseries: []mimirpb.PreallocTimeseries{
				{TimeSeries: &mimirpb.TimeSeries{Labels: []mimirpb.LabelAdapter{{Name: labels.MetricName, Value: "s"}, {Name: "l", Value: "1"}}, Samples: s1}},
				{TimeSeries: &mimirpb.TimeSeries{Labels: []mimirpb.LabelAdapter{{Name: labels.MetricName, Value: "s"}, {Name: "l", Value: "2"}}, Samples: s1}}},
			Source: mimirpb.API,
		})
		require.NoError(t, err)
	}

	// Push data to blocks ingester.
	{
		ingesterBlocksClient, err := ingester_client.MakeIngesterClient(ingesterBlocks.GRPCEndpoint(), clientConfig)
		require.NoError(t, err)
		defer ingesterBlocksClient.Close()

		_, err = ingesterBlocksClient.Push(user.InjectOrgID(context.Background(), "user"), &mimirpb.WriteRequest{
			Timeseries: []mimirpb.PreallocTimeseries{
				{TimeSeries: &mimirpb.TimeSeries{Labels: []mimirpb.LabelAdapter{{Name: labels.MetricName, Value: "s"}, {Name: "l", Value: "2"}}, Samples: s2}},
				{TimeSeries: &mimirpb.TimeSeries{Labels: []mimirpb.LabelAdapter{{Name: labels.MetricName, Value: "s"}, {Name: "l", Value: "3"}}, Samples: s1}}},
			Source: mimirpb.API,
		})
		require.NoError(t, err)
	}

	c, err := e2emimir.NewClient("", querier.HTTPEndpoint(), "", "", "user")
	require.NoError(t, err)

	// Query back the series (1 only in the storage, 1 only in the ingesters, 1 on both).
	result, err := c.Query("s[1m]", time.Unix(10, 0))
	require.NoError(t, err)

	s1Values := []model.SamplePair{
		{Value: 1, Timestamp: 1000},
		{Value: 2, Timestamp: 2000},
		{Value: 3, Timestamp: 3000},
		{Value: 4, Timestamp: 4000},
		{Value: 5, Timestamp: 5000},
	}

	s1AndS2ValuesMerged := []model.SamplePair{
		{Value: 1, Timestamp: 1000},
		{Value: 2, Timestamp: 2000},
		{Value: 2.5, Timestamp: 2500},
		{Value: 3, Timestamp: 3000},
		{Value: 4, Timestamp: 4000},
		{Value: 5, Timestamp: 5000},
		{Value: 5.5, Timestamp: 5500},
	}

	expectedMatrix := model.Matrix{
		// From chunks ingester only.
		&model.SampleStream{
			Metric: model.Metric{labels.MetricName: "s", "l": "1"},
			Values: s1Values,
		},

		// From blocks ingester only.
		&model.SampleStream{
			Metric: model.Metric{labels.MetricName: "s", "l": "3"},
			Values: s1Values,
		},

		// Merged from both ingesters.
		&model.SampleStream{
			Metric: model.Metric{labels.MetricName: "s", "l": "2"},
			Values: s1AndS2ValuesMerged,
		},
	}

	require.Equal(t, model.ValMatrix, result.Type())
	require.ElementsMatch(t, expectedMatrix, result.(model.Matrix))
}
