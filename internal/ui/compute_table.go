package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/rebash-rebash/g9s/internal/model"
)

// vmTableDelegate keeps the Compute Engine list dense and aligned like a
// terminal operations tool. The actual terminal width is applied by the
// parent model; rows never introduce dashboard-style cards.
type vmTableDelegate struct{}

func (d vmTableDelegate) Height() int { return 1 }
func (d vmTableDelegate) Spacing() int { return 0 }
func (d vmTableDelegate) Update(_ interface{}, _ *list.Model) {}

func vmTableHeader(width int) string {
	name, zone, machine, status := vmColumnWidths(width)
	return fmt.Sprintf("%-*s  %-*s  %-*s  %s", name, "NAME", zone, "ZONE", machine, "MACHINE TYPE", "STATUS")
}

func vmRow(vm model.VM, width int) string {
	name, zone, machine, _ := vmColumnWidths(width)
	return fmt.Sprintf("%-*s  %-*s  %-*s  %s", name, truncate(vm.Name, name), zone, truncate(vm.Zone, zone), machine, truncate(vm.MachineType, machine), statusStyle(vm.Status).Render(vm.Status))
}

func vmColumnWidths(width int) (int, int, int, int) {
	available := width - 6
	if available < 40 {
		available = 40
	}
	name := available * 38 / 100
	zone := available * 20 / 100
	machine := available * 25 / 100
	status := available - name - zone - machine
	if name < 18 { name = 18 }
	if zone < 12 { zone = 12 }
	if machine < 16 { machine = 16 }
	if status < 10 { status = 10 }
	return name, zone, machine, status
}
