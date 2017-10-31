package xlog

import (
	"context"
	"fmt"
	"sync"

	"github.com/gravitational/trace"

	"cloud.google.com/go/bigquery"
)

type ProgressReporter struct {
	uploader *bigquery.Uploader
}

var reporters sync.Map

// NewProgressReporter initializes progress reporter
func NewProgressReporter(ctx context.Context, projectID, datasetID, tableID string) (*ProgressReporter, error) {
	key := fmt.Sprintf("%s-%s-%s", projectID, datasetID, tableID)
	stored, ok := reporters.Load(key)
	if ok {
		return stored.(*ProgressReporter), nil
	}

	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, trace.ConvertSystemError(err)
	}

	rep := ProgressReporter{
		uploader: client.Dataset(datasetID).Table(tableID).Uploader(),
	}

	reporters.Store(key, &rep)
	return &rep, nil
}

func (r *ProgressReporter) Put(ctx context.Context, record interface{}) error {
	return trace.Wrap(r.uploader.Put(ctx, record))
}
