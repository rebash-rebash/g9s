package gcp

import (
	"context"
	"fmt"

	"golang.org/x/oauth2/google"
)

// Client contains the shared GCP context used by resource services.
type Client struct {
	ProjectID string
}

// NewClient creates a read-only GCP client using Application Default Credentials.
// An explicit project ID takes precedence over the project discovered from ADC.
func NewClient(ctx context.Context, projectID string) (*Client, error) {
	if projectID == "" {
		creds, err := google.FindDefaultCredentials(ctx)
		if err != nil {
			return nil, fmt.Errorf("find Google Application Default Credentials: %w", err)
		}
		projectID = creds.ProjectID
	}

	if projectID == "" {
		return nil, fmt.Errorf("GCP project ID was not found; set GOOGLE_CLOUD_PROJECT or configure ADC with a project")
	}

	return &Client{ProjectID: projectID}, nil
}
