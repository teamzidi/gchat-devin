import os
from typing import Optional
from google.cloud import secretmanager


class SecretManagerClient:
    """Client for interacting with Google Cloud Secret Manager."""

    def __init__(self, project_id: Optional[str] = None):
        """Initialize the Secret Manager client.

        Args:
            project_id: The Google Cloud project ID. If not provided, it will be
                retrieved from the GOOGLE_CLOUD_PROJECT environment variable.
        """
        self.project_id = project_id or os.environ.get("GOOGLE_CLOUD_PROJECT")
        if not self.project_id:
            raise ValueError(
                "Project ID must be provided or set in GOOGLE_CLOUD_PROJECT environment variable"
            )
        self.client = secretmanager.SecretManagerServiceClient()

    def get_secret(self, secret_id: str, version_id: str = "latest") -> str:
        """Get a secret from Secret Manager.

        Args:
            secret_id: The ID of the secret.
            version_id: The version of the secret. Defaults to "latest".

        Returns:
            The secret value as a string.
        """
        name = f"projects/{self.project_id}/secrets/{secret_id}/versions/{version_id}"
        response = self.client.access_secret_version(request={"name": name})
        return response.payload.data.decode("UTF-8")
