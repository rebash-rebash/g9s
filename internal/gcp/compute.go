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

// ComputeService provides read-only Compute Engine operations.
type ComputeService struct {
	instances *compute.InstancesClient
	projectID string
}

func NewComputeService(ctx context.Context, client *Client) (*ComputeService, error) {
	instances, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create Compute Engine client: %w", err)
	}
	return &ComputeService{instances: instances, projectID: client.ProjectID}, nil
}

func (s *ComputeService) Close() error { return s.instances.Close() }

// ListVMs returns all Compute Engine VMs in the selected project.
func (s *ComputeService) ListVMs(ctx context.Context) ([]model.VM, error) {
	client := s.instances.AggregatedList(ctx, &computepb.AggregatedListInstancesRequest{
		Project: s.projectID,
	})

	var vms []model.VM
	for {
		scope, err := client.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list Compute Engine instances: %w", err)
		}
		if scope.Value == nil {
			continue
		}
		for _, instance := range scope.Value.Instances {
			vms = append(vms, normalizeVM(instance))
		}
	}
	return vms, nil
}

func normalizeVM(instance *computepb.Instance) model.VM {
	vm := model.VM{
		Name:         instance.GetName(),
		InstanceID:   fmt.Sprintf("%d", instance.GetId()),
		Status:       instance.GetStatus(),
		CreationTime: instance.GetCreationTimestamp(),
		DiskCount:    len(instance.GetDisks()),
		NetworkCount: len(instance.GetNetworkInterfaces()),
	}
	if instance.GetZone() != "" {
		parts := strings.Split(instance.GetZone(), "/")
		vm.Zone = parts[len(parts)-1]
	}
	if instance.GetMachineType() != "" {
		parts := strings.Split(instance.GetMachineType(), "/")
		vm.MachineType = parts[len(parts)-1]
	}
	for _, nic := range instance.GetNetworkInterfaces() {
		if vm.InternalIP == "" {
			vm.InternalIP = nic.GetNetworkIP()
		}
		if vm.ExternalIP == "" {
			for _, access := range nic.GetAccessConfigs() {
				if access.GetNatIP() != "" {
					vm.ExternalIP = access.GetNatIP()
					break
				}
			}
		}
	}
	for _, disk := range instance.GetDisks() {
		if disk.GetBoot() {
			vm.BootDisk = disk.GetSource()
			parts := strings.Split(vm.BootDisk, "/")
			vm.BootDisk = parts[len(parts)-1]
			break
		}
	}
	return vm
}
