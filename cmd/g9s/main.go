package main

import (
	"context"
	"fmt"

	"github.com/rebash-rebash/g9s/internal/gcp"
)

func main() {
	ctx := context.Background()
	client, err := gcp.NewClient(ctx, "")
	if err != nil {
		panic(err)
	}
	fmt.Printf("g9s — GCP Resource & Cost Explorer (%s)\n", client.ProjectID)
}
