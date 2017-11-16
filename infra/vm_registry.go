package infra

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/gravitational/trace"

	"cloud.google.com/go/datastore"
	log "github.com/sirupsen/logrus"
)

//gcsDatastoreVmRegistry stores VM registry records inside google datastore
type dsVmRegistry struct {
	client *datastore.Client
	logger log.FieldLogger
}

type dsVmEntry struct {
	Enabled       bool
	Cloud         string
	Checkpoint    string
	Region        string
	ResourceGroup string
	Param         string         `datastore:"param,noindex"`
	K             *datastore.Key `datastore:"__key__"`
}

const (
	vmEntryKind = "VmImage"
)

func GCSDatastoreVmRegistry(ctx context.Context, projectID string, logger log.FieldLogger) (VmRegistry, error) {
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return &dsVmRegistry{client, logger}, nil
}

// Locate returns VM image or trace.NotFound when none was found
func (r *dsVmRegistry) Locate(ctx context.Context, cloud, checkpoint string, param interface{}) (*VmImage, error) {
	var entries []dsVmEntry
	q := datastore.NewQuery(vmEntryKind)
	_, err := r.client.GetAll(ctx, q, &entries)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	// serialize and deserialize back params for comparison
	var paramGeneralized map[string]interface{}
	data, err := json.Marshal(param)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	err = json.Unmarshal(data, &paramGeneralized)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	for _, rec := range entries {
		if rec.Cloud != cloud ||
			rec.Checkpoint != checkpoint ||
			!rec.Enabled {
			continue
		}

		err = compareJSON(paramGeneralized, rec.Param)
		r.logger.WithError(err).Info(rec.Param)
		if err == nil {
			return &VmImage{
				Cloud:         rec.Cloud,
				Region:        rec.Region,
				ResourceGroup: rec.ResourceGroup,
			}, nil
		}
		if trace.IsCompareFailed(err) {
			continue
		}
	}

	return nil, trace.NotFound("VM image not found for cloud=%q, checkpoint=%q, param=%+v", cloud, checkpoint, param)
}

func compareJSON(orig map[string]interface{}, encoded string) error {
	var param map[string]interface{}
	err := json.Unmarshal([]byte(encoded), &param)
	if err != nil {
		return trace.Wrap(err)
	}
	if reflect.DeepEqual(orig, param) {
		return nil
	}
	return trace.CompareFailed("not equal")
}

func (r *dsVmRegistry) Store(ctx context.Context, checkpoint string, param interface{}, image VmImage) error {
	data, err := json.Marshal(param)
	if err != nil {
		return trace.Wrap(err)
	}

	_, err = r.client.Put(ctx,
		datastore.IncompleteKey(vmEntryKind, nil),
		&dsVmEntry{
			Enabled:       true,
			Checkpoint:    checkpoint,
			Cloud:         image.Cloud,
			Region:        image.Region,
			ResourceGroup: image.ResourceGroup,
			Param:         string(data),
		})
	return trace.Wrap(err)
}