// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/cortexproject/cortex/blob/master/pkg/querier/queryrange/value_test.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Cortex Authors.

package queryrange

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql"
	"github.com/stretchr/testify/require"

	"github.com/grafana/mimir/pkg/mimirpb"
)

func TestFromValue(t *testing.T) {
	var testExpr = []struct {
		input    *promql.Result
		err      bool
		expected []SampleStream
	}{
		// string (errors)
		{
			input: &promql.Result{Value: promql.String{T: 1, V: "hi"}},
			err:   true,
		},
		{
			input: &promql.Result{Err: errors.New("foo")},
			err:   true,
		},
		// Scalar
		{
			input: &promql.Result{Value: promql.Scalar{T: 1, V: 1}},
			err:   false,
			expected: []SampleStream{
				{
					Samples: []mimirpb.Sample{
						{
							Value:       1,
							TimestampMs: 1,
						},
					},
				},
			},
		},
		// Vector
		{
			input: &promql.Result{
				Value: promql.Vector{
					promql.Sample{
						Point: promql.Point{T: 1, V: 1},
						Metric: labels.Labels{
							{Name: "a", Value: "a1"},
							{Name: "b", Value: "b1"},
						},
					},
					promql.Sample{
						Point: promql.Point{T: 2, V: 2},
						Metric: labels.Labels{
							{Name: "a", Value: "a2"},
							{Name: "b", Value: "b2"},
						},
					},
				},
			},
			err: false,
			expected: []SampleStream{
				{
					Labels: []mimirpb.LabelAdapter{
						{Name: "a", Value: "a1"},
						{Name: "b", Value: "b1"},
					},
					Samples: []mimirpb.Sample{
						{
							Value:       1,
							TimestampMs: 1,
						},
					},
				},
				{
					Labels: []mimirpb.LabelAdapter{
						{Name: "a", Value: "a2"},
						{Name: "b", Value: "b2"},
					},
					Samples: []mimirpb.Sample{
						{
							Value:       2,
							TimestampMs: 2,
						},
					},
				},
			},
		},
		// Matrix
		{
			input: &promql.Result{
				Value: promql.Matrix{
					{
						Metric: labels.Labels{
							{Name: "a", Value: "a1"},
							{Name: "b", Value: "b1"},
						},
						Points: []promql.Point{
							{T: 1, V: 1},
							{T: 2, V: 2},
						},
					},
					{
						Metric: labels.Labels{
							{Name: "a", Value: "a2"},
							{Name: "b", Value: "b2"},
						},
						Points: []promql.Point{
							{T: 1, V: 8},
							{T: 2, V: 9},
						},
					},
				},
			},
			err: false,
			expected: []SampleStream{
				{
					Labels: []mimirpb.LabelAdapter{
						{Name: "a", Value: "a1"},
						{Name: "b", Value: "b1"},
					},
					Samples: []mimirpb.Sample{
						{
							Value:       1,
							TimestampMs: 1,
						},
						{
							Value:       2,
							TimestampMs: 2,
						},
					},
				},
				{
					Labels: []mimirpb.LabelAdapter{
						{Name: "a", Value: "a2"},
						{Name: "b", Value: "b2"},
					},
					Samples: []mimirpb.Sample{
						{
							Value:       8,
							TimestampMs: 1,
						},
						{
							Value:       9,
							TimestampMs: 2,
						},
					},
				},
			},
		},
	}

	for i, c := range testExpr {
		t.Run(fmt.Sprintf("[%d]", i), func(t *testing.T) {
			result, err := FromResult(c.input)
			if c.err {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
				require.Equal(t, c.expected, result)
			}
		})
	}
}
