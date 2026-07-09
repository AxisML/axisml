package metricsquery_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/metricsquery"
	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
)

// QueryRange failure surfaces as CodeUnavailable (query.go:84-86).
func TestSeries_ProviderError_Unavailable(t *testing.T) {
	fp := &fakeProvider{enabled: true, err: errors.New("prometheus down")}
	_, err := metricsquery.NewQuerier(fp).
		Series(context.Background(), "acme", []string{"p"}, "cpu_util", "1h", "")
	require.Error(t, err)
	assert.Equal(t, apperrors.CodeUnavailable, codeOf(t, err))
	assert.True(t, fp.called, "should have attempted the query")
}

// A malformed day/week window is a validation error (parseWindow strconv paths).
func TestSeries_BadDayWeekRange_Validation(t *testing.T) {
	fp := &fakeProvider{enabled: true}
	q := metricsquery.NewQuerier(fp)
	for _, bad := range []string{"xd", "xw", "1.5d"} {
		_, err := q.Series(context.Background(), "acme", []string{"p"}, "cpu_util", bad, "")
		require.Error(t, err, "range %s", bad)
		assert.Equal(t, apperrors.CodeValidation, codeOf(t, err), "range %s", bad)
		assert.False(t, fp.called, "must reject before querying, range %s", bad)
	}
}

// A malformed explicit step is a validation error (resolveStep parseWindow path).
func TestSeries_BadExplicitStep_Validation(t *testing.T) {
	fp := &fakeProvider{enabled: true}
	for _, bad := range []string{"abc", "xd"} {
		_, err := metricsquery.NewQuerier(fp).
			Series(context.Background(), "acme", []string{"p"}, "cpu_util", "1h", bad)
		require.Error(t, err, "step %s", bad)
		assert.Equal(t, apperrors.CodeValidation, codeOf(t, err), "step %s", bad)
		assert.False(t, fp.called, "must reject before querying, step %s", bad)
	}
}

// A valid day/week window with derived step succeeds and clamps step to <=10m.
func TestSeries_DerivedStepClampedForLongRange(t *testing.T) {
	fp := &fakeProvider{enabled: true}
	s, err := metricsquery.NewQuerier(fp).
		Series(context.Background(), "acme", []string{"p"}, "cpu_util", "30d", "")
	require.NoError(t, err)
	// 30d/50 far exceeds 10m so the derived step clamps to the 10m ceiling.
	assert.Equal(t, "10m0s", s.Step)
}

func TestPodNames(t *testing.T) {
	tests := []struct {
		name string
		list *corev1.PodList
		want []string
	}{
		{name: "nil list", list: nil, want: nil},
		{name: "empty list", list: &corev1.PodList{}, want: []string{}},
		{
			name: "filters blank names",
			list: &corev1.PodList{Items: []corev1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "p1"}},
				{ObjectMeta: metav1.ObjectMeta{Name: ""}},
				{ObjectMeta: metav1.ObjectMeta{Name: "p2"}},
			}},
			want: []string{"p1", "p2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, metricsquery.PodNames(tt.list))
		})
	}
}
