package backup

import (
	"context"
	"fmt"

	//nolint:staticcheck // transfermanager is the successor but isn't GA yet on the SDK we pin
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client implements Uploader by combining the standard PutObject (used for
// the small fallback artifact) with a multipart manager.Uploader (used for
// streaming the tar pipe).
type S3Client struct {
	*s3.Client
	multipart *manager.Uploader //nolint:staticcheck // see import comment
}

// NewS3Client wraps an *s3.Client with the multipart manager.
func NewS3Client(c *s3.Client) *S3Client {
	//nolint:staticcheck // see import comment
	return &S3Client{Client: c, multipart: manager.NewUploader(c)}
}

// StreamUpload sends an unbounded io.Reader as a multipart S3 upload.
func (c *S3Client) StreamUpload(ctx context.Context, in *s3.PutObjectInput) error {
	//nolint:staticcheck // see import comment
	if _, err := c.multipart.Upload(ctx, in); err != nil {
		return fmt.Errorf("multipart upload: %w", err)
	}
	return nil
}
