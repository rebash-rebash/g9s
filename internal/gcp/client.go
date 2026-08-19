package gcp

import "context"

type Client struct {
	ProjectID string
}

func NewClient(ctx context.Context, projectID string) (*Client, error) {
	return &Client{ProjectID: projectID}, nil
}
