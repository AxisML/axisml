// Package s3 wraps the bits of the S3 protocol that axisml-artifact-hub needs
// against RustFS: GET object (for dataset digest verification at complete time),
// delete-by-prefix (for GC), and bucket bootstrap. It is the S3 analogue of the
// oci package.
//
// MVP simplification (mirrors the oci package): the artifact-hub pod holds
// RustFS admin credentials and signs requests with them directly (SigV4 via
// minio-go). This is NOT scope-limited — acceptable only because RustFS is
// ClusterIP-scoped inside axisml-infra. Phase 2 replaces it with prefix-scoped
// STS sessions.
package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ErrNotFound is returned by GetObject when the key is absent.
var ErrNotFound = errors.New("s3: object not found")

// Client is a thin wrapper around minio-go for the S3 operations artifact-hub
// needs against RustFS.
type Client struct {
	mc     *minio.Client
	bucket string
}

// Config bundles the constructor inputs. Endpoint is the externally reachable
// host:port (e.g. rustfs.axisml-infra:9000); a scheme prefix on Endpoint
// overrides UseSSL.
type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// New returns a Client. A scheme prefix on Endpoint (http:// / https://)
// overrides cfg.UseSSL; otherwise UseSSL selects the scheme.
func New(cfg Config) (*Client, error) {
	endpoint := cfg.Endpoint
	useSSL := cfg.UseSSL
	if strings.HasPrefix(endpoint, "https://") {
		useSSL = true
		endpoint = strings.TrimPrefix(endpoint, "https://")
	} else if strings.HasPrefix(endpoint, "http://") {
		useSSL = false
		endpoint = strings.TrimPrefix(endpoint, "http://")
	}
	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("s3: new client: %w", err)
	}
	return &Client{mc: mc, bucket: cfg.Bucket}, nil
}

// Bucket returns the configured bucket name.
func (c *Client) Bucket() string { return c.bucket }

// EnsureBucket creates the bucket if it does not already exist. Idempotent.
func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("s3: bucket exists %q: %w", c.bucket, err)
	}
	if exists {
		return nil
	}
	if err := c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{}); err != nil {
		// A concurrent creator may have won the race; treat "already owned" as success.
		if minio.ToErrorResponse(err).Code == "BucketAlreadyOwnedByYou" {
			return nil
		}
		return fmt.Errorf("s3: make bucket %q: %w", c.bucket, err)
	}
	return nil
}

// GetObject reads the object at key in full, returning ErrNotFound when the key
// is absent.
func (c *Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	obj, err := c.mc.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("s3: get %q: %w", key, err)
	}
	defer func() { _ = obj.Close() }()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, obj); err != nil {
		if isNoSuchKey(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("s3: read %q: %w", key, err)
	}
	return buf.Bytes(), nil
}

// DeletePrefix removes every object under prefix. NotFound is treated as
// success so GC stays idempotent.
func (c *Client) DeletePrefix(ctx context.Context, prefix string) error {
	objCh := c.mc.ListObjects(ctx, c.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true})
	for rErr := range c.mc.RemoveObjects(ctx, c.bucket, objCh, minio.RemoveObjectsOptions{}) {
		if rErr.Err != nil && !isNoSuchKey(rErr.Err) {
			return fmt.Errorf("s3: remove %q: %w", rErr.ObjectName, rErr.Err)
		}
	}
	return nil
}

// isNoSuchKey reports whether err is an S3 "key/bucket absent" response.
func isNoSuchKey(err error) bool {
	code := minio.ToErrorResponse(err).Code
	return code == "NoSuchKey" || code == "NoSuchBucket"
}
