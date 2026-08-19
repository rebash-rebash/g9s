package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rebash-rebash/g9s/internal/gcp"
	"github.com/rebash-rebash/g9s/internal/ui"
)

func main() {
	ctx := context.Background()
	client, err := gcp.NewClient(ctx, os.Getenv("GOOGLE_CLOUD_PROJECT"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "g9s: %v\n", err)
		os.Exit(1)
	}

	computeSvc, err := gcp.NewComputeService(ctx, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "g9s: %v\n", err)
		os.Exit(1)
	}
	defer computeSvc.Close()

	monitoringSvc, err := gcp.NewMonitoringService(ctx, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "g9s: %v\n", err)
		os.Exit(1)
	}
	defer monitoringSvc.Close()

	p := tea.NewProgram(
		ui.New(client.ProjectID, computeSvc, monitoringSvc),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "g9s: %v\n", err)
		os.Exit(1)
	}
}
