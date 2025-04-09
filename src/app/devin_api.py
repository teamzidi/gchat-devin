import os
import requests
from typing import Dict, Any, Optional


class DevinAPIClient:
    """Client for interacting with the Devin API."""

    def __init__(self, api_key: str, base_url: str = "https://api.devin.ai/v1"):
        """Initialize the Devin API client.

        Args:
            api_key: The Devin API key.
            base_url: The base URL for the Devin API.
        """
        self.api_key = api_key
        self.base_url = base_url
        self.headers = {
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        }

    def create_session(self, prompt: str) -> Dict[str, Any]:
        """Create a new Devin session.

        Args:
            prompt: The initial prompt for Devin.

        Returns:
            The session details including session_id and URL.
        """
        url = f"{self.base_url}/sessions"
        payload = {"prompt": prompt}
        
        response = requests.post(url, headers=self.headers, json=payload)
        response.raise_for_status()
        
        return response.json()

    def send_message(self, session_id: str, message: str) -> Dict[str, Any]:
        """Send a message to an existing Devin session.

        Args:
            session_id: The ID of the Devin session.
            message: The message to send to Devin.

        Returns:
            The response from the API.
        """
        url = f"{self.base_url}/session/{session_id}/message"
        payload = {"message": message}
        
        response = requests.post(url, headers=self.headers, json=payload)
        response.raise_for_status()
        
        return response.json()

    def get_session_details(self, session_id: str) -> Dict[str, Any]:
        """Get details about an existing Devin session.

        Args:
            session_id: The ID of the Devin session.

        Returns:
            The session details.
        """
        url = f"{self.base_url}/session/{session_id}"
        
        response = requests.get(url, headers=self.headers)
        response.raise_for_status()
        
        return response.json()

    def get_enterprise_consumption(self) -> Dict[str, Any]:
        """Get enterprise consumption data.

        Returns:
            The enterprise consumption data.
        """
        url = f"{self.base_url}/enterprise/consumption"
        
        response = requests.get(url, headers=self.headers)
        response.raise_for_status()
        
        return response.json()
