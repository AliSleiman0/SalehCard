package blob

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
)

// AzureStorage uploads to an Azure Blob Storage container over a connection
// string (no managed identity in v1 — see PLAN-PRODUCT-IMAGES.md). The
// container must already exist with public blob-level read access; the
// production container is provisioned once from Azure Cloud Shell (see
// DEVOPS-TODO.md).
type AzureStorage struct {
	client    *azblob.Client
	container string
	// serviceURL is the account's blob endpoint, used to build public URLs
	// without an extra round trip.
	serviceURL string
}

// newAzureStorage builds an Azure-backed Storage from cfg, erroring if the
// connection string or container is missing, or the connection string doesn't
// parse.
func newAzureStorage(cfg AzureConfig) (Storage, error) {
	if cfg.ConnectionString == "" || cfg.Container == "" {
		return nil, fmt.Errorf("azure: connection string and container are required")
	}
	client, err := azblob.NewClientFromConnectionString(cfg.ConnectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: client: %w", err)
	}
	return &AzureStorage{
		client:     client,
		container:  cfg.Container,
		serviceURL: client.URL(),
	}, nil
}

// Upload writes data as a block blob at key, setting the HTTP content type,
// and returns the blob's public URL.
func (a *AzureStorage) Upload(ctx context.Context, key, contentType string, data []byte) (string, error) {
	_, err := a.client.UploadBuffer(ctx, a.container, key, data, &azblob.UploadBufferOptions{
		HTTPHeaders: &blob.HTTPHeaders{BlobContentType: &contentType},
	})
	if err != nil {
		return "", fmt.Errorf("azure: upload: %w", err)
	}
	base, err := url.Parse(a.serviceURL)
	if err != nil {
		return "", fmt.Errorf("azure: parse service url: %w", err)
	}
	base.Path = fmt.Sprintf("/%s/%s", a.container, key)
	return base.String(), nil
}

// Delete removes the blob at key. A blob that does not exist is treated as
// success (idempotent per the [Storage] contract), so purge retries never fail
// on already-deleted objects.
func (a *AzureStorage) Delete(ctx context.Context, key string) error {
	if _, err := a.client.DeleteBlob(ctx, a.container, key, nil); err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return nil
		}
		return fmt.Errorf("azure: delete: %w", err)
	}
	return nil
}
