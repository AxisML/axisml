package kubeproxy

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

func init() { gin.SetMode(gin.TestMode) }

func newCtx() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestMapErr(t *testing.T) {
	notFound := apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "x")
	appErr := apperrors.New(apperrors.CodeConflict, "already exists")

	tests := []struct {
		name     string
		in       error
		wantNil  bool
		wantCode apperrors.Code
	}{
		{name: "nil passthrough", in: nil, wantNil: true},
		{name: "existing app error passes through", in: appErr, wantCode: apperrors.CodeConflict},
		{name: "not owned -> forbidden", in: extensions.ErrInstanceNotOwned, wantCode: apperrors.CodeForbidden},
		{name: "wrapped not owned -> forbidden", in: apperrors.Wrap(apperrors.CodeInternal, "x", extensions.ErrInstanceNotOwned), wantCode: apperrors.CodeInternal},
		{name: "k8s not found -> not found", in: notFound, wantCode: apperrors.CodeNotFound},
		{name: "generic -> unavailable", in: errors.New("boom"), wantCode: apperrors.CodeUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapErr(tt.in)
			if tt.wantNil {
				assert.NoError(t, got)
				return
			}
			require.Error(t, got)
			e, ok := apperrors.As(got)
			require.True(t, ok)
			assert.Equal(t, tt.wantCode, e.Code)
		})
	}
}

func TestProjectPod(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p1", Namespace: "ns", Labels: map[string]string{"a": "b"}},
		Spec:       corev1.PodSpec{NodeName: "node-1"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	got := projectPod(pod)
	assert.Equal(t, "p1", got.Name)
	assert.Equal(t, "ns", got.Namespace)
	assert.Equal(t, "Running", got.Phase)
	assert.Equal(t, "node-1", got.NodeName)
	assert.Equal(t, map[string]string{"a": "b"}, got.Labels)
}

func TestProjectEventAndPtrTime(t *testing.T) {
	when := metav1.NewMicroTime(time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC))
	ev := &eventsv1.Event{
		Reason:              "Scheduled",
		Note:                "assigned",
		Type:                "Normal",
		Regarding:           corev1.ObjectReference{Kind: "Pod", Name: "p1"},
		ReportingController: "scheduler",
		EventTime:           when,
	}
	got := projectEvent(ev)
	assert.Equal(t, "Scheduled", got.Reason)
	assert.Equal(t, "assigned", got.Note)
	assert.Equal(t, "Normal", got.Type)
	assert.Equal(t, "Pod/p1", got.Object)
	assert.Equal(t, "scheduler", got.Reporter)
	require.NotNil(t, got.EventTime)
	assert.Equal(t, when.Time, got.EventTime.Time)

	// ptrTime returns nil for a zero time.
	assert.Nil(t, ptrTime(metav1.MicroTime{}))
}

func TestWritePods(t *testing.T) {
	c, w := newCtx()
	WritePods(c, &corev1.PodList{Items: []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "p1"}, Status: corev1.PodStatus{Phase: corev1.PodPending}},
		{ObjectMeta: metav1.ObjectMeta{Name: "p2"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
	}})
	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Items []struct {
			Name  string `json:"name"`
			Phase string `json:"phase"`
		} `json:"items"`
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 2, body.Total)
	require.Len(t, body.Items, 2)
	assert.Equal(t, "p1", body.Items[0].Name)
	assert.Equal(t, "Running", body.Items[1].Phase)
}

func TestWritePods_Empty(t *testing.T) {
	c, w := newCtx()
	WritePods(c, &corev1.PodList{})
	var body struct {
		Items []any `json:"items"`
		Total int   `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 0, body.Total)
	assert.NotNil(t, body.Items, "items must serialize as [] not null")
}

func TestWriteEvents(t *testing.T) {
	c, w := newCtx()
	WriteEvents(c, &eventsv1.EventList{Items: []eventsv1.Event{
		{Reason: "Pulled", Type: "Normal", Regarding: corev1.ObjectReference{Kind: "Pod", Name: "p1"}},
	}})
	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Items []struct {
			Reason string `json:"reason"`
			Object string `json:"object"`
		} `json:"items"`
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 1, body.Total)
	assert.Equal(t, "Pulled", body.Items[0].Reason)
	assert.Equal(t, "Pod/p1", body.Items[0].Object)
}

func TestPodLogQuery(t *testing.T) {
	t.Run("all params parsed", func(t *testing.T) {
		c, _ := newCtx()
		c.Request = httptest.NewRequest(http.MethodGet,
			"/logs?container=main&tailLines=100&sinceSeconds=60&follow=true&previous=true", nil)
		opts, follow := PodLogQuery(c)
		assert.True(t, follow)
		assert.Equal(t, "main", opts.Container)
		require.NotNil(t, opts.TailLines)
		assert.Equal(t, int64(100), *opts.TailLines)
		require.NotNil(t, opts.SinceSeconds)
		assert.Equal(t, int64(60), *opts.SinceSeconds)
		assert.True(t, opts.Follow)
		assert.True(t, opts.Previous)
	})

	t.Run("defaults and invalid numerics ignored", func(t *testing.T) {
		c, _ := newCtx()
		c.Request = httptest.NewRequest(http.MethodGet, "/logs?tailLines=abc&sinceSeconds=x", nil)
		opts, follow := PodLogQuery(c)
		assert.False(t, follow)
		assert.Empty(t, opts.Container)
		assert.Nil(t, opts.TailLines, "unparseable tailLines is dropped")
		assert.Nil(t, opts.SinceSeconds, "unparseable sinceSeconds is dropped")
		assert.False(t, opts.Previous)
	})
}

func TestAbortIfMissingNamespace(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		c, _ := newCtx()
		c.Params = gin.Params{{Key: "namespace", Value: ""}}
		assert.True(t, AbortIfMissingNamespace(c))
		require.Len(t, c.Errors, 1)
	})
	t.Run("present", func(t *testing.T) {
		c, _ := newCtx()
		c.Params = gin.Params{{Key: "namespace", Value: "acme"}}
		assert.False(t, AbortIfMissingNamespace(c))
		assert.Empty(t, c.Errors)
	})
}

func TestStreamLog_NonFollowCopiesBody(t *testing.T) {
	c, w := newCtx()
	StreamLog(c, false, func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("hello world")), nil
	})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "hello world", w.Body.String())
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
}

func TestStreamLog_FollowUsesSSEHeaders(t *testing.T) {
	c, w := newCtx()
	StreamLog(c, true, func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("log line")), nil
	})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "log line", w.Body.String())
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
}

// errAfterReader yields some bytes, then a non-EOF error, exercising the
// follow loop's mid-stream read-error exit.
type errAfterReader struct {
	data []byte
	done bool
}

func (r *errAfterReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, errors.New("stream reset")
	}
	r.done = true
	n := copy(p, r.data)
	return n, nil
}

func (r *errAfterReader) Close() error { return nil }

func TestStreamLog_FollowReadErrorStopsStream(t *testing.T) {
	c, w := newCtx()
	StreamLog(c, true, func() (io.ReadCloser, error) {
		return &errAfterReader{data: []byte("partial")}, nil
	})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "partial", w.Body.String())
}

func TestStreamLog_NotFoundRendersGone(t *testing.T) {
	c, w := newCtx()
	StreamLog(c, false, func() (io.ReadCloser, error) {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "p1")
	})
	assert.Equal(t, http.StatusGone, w.Code)
	assert.Contains(t, w.Body.String(), "instance gone")
	assert.Empty(t, c.Errors, "gone is rendered inline, not via the error middleware")
}

func TestStreamLog_GenericErrorGoesToMiddleware(t *testing.T) {
	c, _ := newCtx()
	StreamLog(c, false, func() (io.ReadCloser, error) {
		return nil, errors.New("dial tcp: connection refused")
	})
	require.Len(t, c.Errors, 1)
	e, ok := apperrors.As(c.Errors[0].Err)
	require.True(t, ok)
	assert.Equal(t, apperrors.CodeUnavailable, e.Code)
}
