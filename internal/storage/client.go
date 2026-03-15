// Package storage provides an abstraction over the orchestrator's storage
// protocol for reading and writing API collections, environments, and history
// as markdown files with YAML frontmatter metadata.
package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/helpers"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	CollectionsDir  = "api/collections"
	EnvironmentsDir = "api/environments"
	HistoryDir      = "api/history"
	GlobalProject   = ".global"
)

// StorageClient is the interface for communicating with the orchestrator.
type StorageClient interface {
	Send(ctx context.Context, req *pluginv1.PluginRequest) (*pluginv1.PluginResponse, error)
}

// DataStorage provides high-level operations for API data.
type DataStorage struct {
	client StorageClient
}

// NewDataStorage creates a new DataStorage backed by the given client.
func NewDataStorage(client StorageClient) *DataStorage {
	return &DataStorage{client: client}
}

// ResolveProject returns the project slug, defaulting to GlobalProject.
func ResolveProject(projectID string) string {
	if projectID == "" {
		return GlobalProject
	}
	return projectID
}

// ---------- Collections ----------

func collectionPath(project, colID string) string {
	return filepath.Join(project, CollectionsDir, colID+".md")
}

func (ds *DataStorage) ReadCollection(ctx context.Context, project, colID string) (*structpb.Struct, string, int64, error) {
	path := collectionPath(project, colID)
	resp, err := ds.storageRead(ctx, path)
	if err != nil {
		return nil, "", 0, fmt.Errorf("read collection %s: %w", colID, err)
	}
	return resp.Metadata, string(resp.Content), resp.Version, nil
}

func (ds *DataStorage) WriteCollection(ctx context.Context, project, colID string, metadata *structpb.Struct, body string, expectedVersion int64) (int64, error) {
	path := collectionPath(project, colID)
	return ds.storageWrite(ctx, path, metadata, []byte(body), expectedVersion)
}

func (ds *DataStorage) DeleteCollection(ctx context.Context, project, colID string) error {
	path := collectionPath(project, colID)
	return ds.storageDelete(ctx, path)
}

func (ds *DataStorage) ListCollections(ctx context.Context, project string) ([]*pluginv1.StorageEntry, error) {
	prefix := filepath.Join(project, CollectionsDir) + string(filepath.Separator)
	return ds.storageList(ctx, prefix, "*.md")
}

// IDFromPath extracts the ID from a storage entry path.
func IDFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".md")
}

// ---------- Environments ----------

func envPath(project, name string) string {
	return filepath.Join(project, EnvironmentsDir, name+".md")
}

func (ds *DataStorage) ReadEnvironment(ctx context.Context, project, name string) (*structpb.Struct, string, int64, error) {
	path := envPath(project, name)
	resp, err := ds.storageRead(ctx, path)
	if err != nil {
		return nil, "", 0, fmt.Errorf("read environment %s: %w", name, err)
	}
	return resp.Metadata, string(resp.Content), resp.Version, nil
}

func (ds *DataStorage) WriteEnvironment(ctx context.Context, project, name string, metadata *structpb.Struct, body string, expectedVersion int64) (int64, error) {
	path := envPath(project, name)
	return ds.storageWrite(ctx, path, metadata, []byte(body), expectedVersion)
}

func (ds *DataStorage) ListEnvironments(ctx context.Context, project string) ([]*pluginv1.StorageEntry, error) {
	prefix := filepath.Join(project, EnvironmentsDir) + string(filepath.Separator)
	return ds.storageList(ctx, prefix, "*.md")
}

// ---------- History ----------

func historyPath(project, date string) string {
	return filepath.Join(project, HistoryDir, date+".md")
}

func (ds *DataStorage) ReadHistory(ctx context.Context, project, date string) (*structpb.Struct, string, int64, error) {
	path := historyPath(project, date)
	resp, err := ds.storageRead(ctx, path)
	if err != nil {
		return nil, "", 0, fmt.Errorf("read history %s: %w", date, err)
	}
	return resp.Metadata, string(resp.Content), resp.Version, nil
}

func (ds *DataStorage) AppendHistory(ctx context.Context, project, date string, metadata *structpb.Struct, body string, expectedVersion int64) (int64, error) {
	path := historyPath(project, date)
	return ds.storageWrite(ctx, path, metadata, []byte(body), expectedVersion)
}

func (ds *DataStorage) ListHistory(ctx context.Context, project string) ([]*pluginv1.StorageEntry, error) {
	prefix := filepath.Join(project, HistoryDir) + string(filepath.Separator)
	return ds.storageList(ctx, prefix, "*.md")
}

// ---------- Low-level storage protocol ----------

func (ds *DataStorage) storageRead(ctx context.Context, path string) (*pluginv1.StorageReadResponse, error) {
	resp, err := ds.client.Send(ctx, &pluginv1.PluginRequest{
		RequestId: helpers.NewUUID(),
		Request: &pluginv1.PluginRequest_StorageRead{
			StorageRead: &pluginv1.StorageReadRequest{
				Path:        path,
				StorageType: "markdown",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	sr := resp.GetStorageRead()
	if sr == nil {
		return nil, fmt.Errorf("unexpected response type for storage read")
	}
	return sr, nil
}

func (ds *DataStorage) storageWrite(ctx context.Context, path string, metadata *structpb.Struct, content []byte, expectedVersion int64) (int64, error) {
	resp, err := ds.client.Send(ctx, &pluginv1.PluginRequest{
		RequestId: helpers.NewUUID(),
		Request: &pluginv1.PluginRequest_StorageWrite{
			StorageWrite: &pluginv1.StorageWriteRequest{
				Path:            path,
				Content:         content,
				Metadata:        metadata,
				ExpectedVersion: expectedVersion,
				StorageType:     "markdown",
			},
		},
	})
	if err != nil {
		return 0, err
	}
	sw := resp.GetStorageWrite()
	if sw == nil {
		return 0, fmt.Errorf("unexpected response type for storage write")
	}
	if !sw.Success {
		return 0, fmt.Errorf("storage write failed: %s", sw.Error)
	}
	return sw.NewVersion, nil
}

func (ds *DataStorage) storageDelete(ctx context.Context, path string) error {
	resp, err := ds.client.Send(ctx, &pluginv1.PluginRequest{
		RequestId: helpers.NewUUID(),
		Request: &pluginv1.PluginRequest_StorageDelete{
			StorageDelete: &pluginv1.StorageDeleteRequest{
				Path:        path,
				StorageType: "markdown",
			},
		},
	})
	if err != nil {
		return err
	}
	sd := resp.GetStorageDelete()
	if sd == nil {
		return fmt.Errorf("unexpected response type for storage delete")
	}
	if !sd.Success {
		return fmt.Errorf("storage delete failed")
	}
	return nil
}

func (ds *DataStorage) storageList(ctx context.Context, prefix, pattern string) ([]*pluginv1.StorageEntry, error) {
	resp, err := ds.client.Send(ctx, &pluginv1.PluginRequest{
		RequestId: helpers.NewUUID(),
		Request: &pluginv1.PluginRequest_StorageList{
			StorageList: &pluginv1.StorageListRequest{
				Prefix:      prefix,
				Pattern:     pattern,
				StorageType: "markdown",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	sl := resp.GetStorageList()
	if sl == nil {
		return nil, fmt.Errorf("unexpected response type for storage list")
	}
	return sl.Entries, nil
}
