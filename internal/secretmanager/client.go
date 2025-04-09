package secretmanager

import (
	"context"
	"fmt"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

type Client struct {
	ProjectID string
	Client    *secretmanager.Client
}

func NewClient(projectID string) (*Client, error) {
	ctx := context.Background()
	
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
		if projectID == "" {
			return nil, fmt.Errorf("project ID must be provided or set in GOOGLE_CLOUD_PROJECT environment variable")
		}
	}
	
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Secret Manager client: %v", err)
	}
	
	return &Client{
		ProjectID: projectID,
		Client:    client,
	}, nil
}

func (c *Client) GetSecret(ctx context.Context, secretID string, versionID string) (string, error) {
	if versionID == "" {
		versionID = "latest"
	}
	
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", c.ProjectID, secretID, versionID)
	
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}
	
	result, err := c.Client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access secret version: %v", err)
	}
	
	return string(result.Payload.Data), nil
}

func (c *Client) Close() error {
	return c.Client.Close()
}
