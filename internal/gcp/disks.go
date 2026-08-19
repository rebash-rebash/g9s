package gcp

import (
	"context"
	"fmt"
	"strings"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/rebash-rebash/g9s/internal/model"
	"google.golang.org/api/iterator"
)

// DiskService provides read-only Persistent Disk operations.
type DiskService struct {
	disks     *compute.DisksClient
	projectID string
}

func NewDiskService(ctx context.Context, client *Client) (*DiskService, error) {
	disks, err := compute.NewDisksRESTClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create Compute Disk client: %w", err)
	}
	return &DiskService{disks: disks, projectID: client.ProjectID}, nil
}

func (s *DiskService) Close() error { return s.disks.Close() }

// ListDisks returns all zonal Persistent Disks in the selected project.
func (s *DiskService) ListDisks(ctx context.Context) ([]model.Disk, error) {
	it := s.disks.AggregatedList(ctx, &computepb.AggregatedListDisksRequest{Project: s.projectID})
	var disks []model.Disk
	for {
		scope, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list persistent disks: %w", err)
		}
		if scope.Value == nil {
			continue
		}
		for _, disk := range scope.Value.Disks {
			disks = append(disks, normalizeDisk(disk))
		}
	}
	return disks, nil
}

func normalizeDisk(disk *computepb.Disk) model.Disk {
	d := model.Disk{
		Name:     disk.GetName(),
		SizeGB:   disk.GetSizeGb(),
		Attached: len(disk.GetUsers()) > 0,
	}
	if disk.GetZone() != "" {
		parts := strings.Split(disk.GetZone(), "/")
		d.Zone = parts[len(parts)-1]
	}
	return d
}
