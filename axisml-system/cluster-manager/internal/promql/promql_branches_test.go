package promql_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-system/cluster-manager/internal/promql"
)

// emptyProm serves a well-formed but empty matrix so Series exercises the
// len(matrix)==0 projection branch.
func emptyProm() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`))
	}))
}

// errorProm returns a Prometheus API error so Series surfaces the QueryRange
// failure path (neither ErrUnavailable nor ErrBadRequest).
func errorProm() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"status":"error","errorType":"bad_data","error":"boom"}`))
	}))
}

func TestNew_InvalidURL(t *testing.T) {
	_, err := promql.New("http://%zz")
	require.Error(t, err)
}

func TestSeries_EmptyMatrix(t *testing.T) {
	srv := emptyProm()
	defer srv.Close()
	qr, err := promql.New(srv.URL)
	require.NoError(t, err)

	s, err := qr.Series(context.Background(), "acme", nil, "cpu_util", "1h", "")
	require.NoError(t, err)
	assert.Empty(t, s.Series)
	assert.Equal(t, "cores", s.Unit)
}

func TestSeries_QueryError(t *testing.T) {
	srv := errorProm()
	defer srv.Close()
	qr, err := promql.New(srv.URL)
	require.NoError(t, err)

	_, err = qr.Series(context.Background(), "acme", nil, "cpu_util", "1h", "")
	require.Error(t, err)
	// Not a client-side classification: it is the raw downstream error.
	assert.NotErrorIs(t, err, promql.ErrBadRequest)
	assert.NotErrorIs(t, err, promql.ErrUnavailable)
}

// TestSeries_WindowAndStepBranches drives parseWindow (day/week valid + atoi
// errors, empty range) and resolveStep (min/max clamps, explicit-step parse
// error) through the public Series entrypoint.
func TestSeries_WindowAndStepBranches(t *testing.T) {
	srv := emptyProm()
	defer srv.Close()
	qr, err := promql.New(srv.URL)
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("week range derives clamped max step", func(t *testing.T) {
		// 1w/50 ≈ 3.36h → clamped down to the 10m ceiling.
		s, err := qr.Series(ctx, "acme", nil, "cpu_util", "1w", "")
		require.NoError(t, err)
		assert.Equal(t, "10m0s", s.Step)
	})

	t.Run("tiny range derives clamped min step", func(t *testing.T) {
		// 30s/50 = 0.6s → clamped up to the 15s floor.
		s, err := qr.Series(ctx, "acme", nil, "cpu_util", "30s", "")
		require.NoError(t, err)
		assert.Equal(t, "15s", s.Step)
	})

	t.Run("explicit day step is honoured", func(t *testing.T) {
		s, err := qr.Series(ctx, "acme", nil, "cpu_util", "30d", "1d")
		require.NoError(t, err)
		assert.Equal(t, "24h0m0s", s.Step)
	})

	badRange := []string{"", "xd", "xw"}
	for _, r := range badRange {
		t.Run("bad range "+r, func(t *testing.T) {
			_, err := qr.Series(ctx, "acme", nil, "cpu_util", r, "")
			assert.ErrorIs(t, err, promql.ErrBadRequest)
		})
	}

	t.Run("bad explicit step", func(t *testing.T) {
		_, err := qr.Series(ctx, "acme", nil, "cpu_util", "1h", "nonsense")
		assert.ErrorIs(t, err, promql.ErrBadRequest)
	})
}
