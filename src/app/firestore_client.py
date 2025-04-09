import datetime
from typing import Dict, Any, List, Optional
from google.cloud import firestore


class FirestoreClient:
    """Client for interacting with Google Cloud Firestore."""

    def __init__(self, collection_name: str = "gchat_devin_sessions"):
        """Initialize the Firestore client.

        Args:
            collection_name: The name of the Firestore collection to use.
        """
        self.db = firestore.Client()
        self.collection = self.db.collection(collection_name)

    def create_session_mapping(
        self, 
        gchat_space: str, 
        gchat_thread: str, 
        devin_session_id: str, 
        user_id: str,
        prompt: str
    ) -> str:
        """Create a mapping between a Google Chat thread and a Devin session.

        Args:
            gchat_space: The Google Chat space ID.
            gchat_thread: The Google Chat thread ID.
            devin_session_id: The Devin session ID.
            user_id: The ID of the user who created the session.
            prompt: The initial prompt sent to Devin.

        Returns:
            The document ID of the created mapping.
        """
        doc_ref = self.collection.document()
        doc_ref.set({
            "gchat_space": gchat_space,
            "gchat_thread": gchat_thread,
            "devin_session_id": devin_session_id,
            "user_id": user_id,
            "prompt": prompt,
            "created_at": firestore.SERVER_TIMESTAMP,
            "updated_at": firestore.SERVER_TIMESTAMP,
            "messages": [{
                "sender": "user",
                "content": prompt,
                "timestamp": firestore.SERVER_TIMESTAMP
            }]
        })
        return doc_ref.id

    def get_session_by_thread(
        self, gchat_space: str, gchat_thread: str
    ) -> Optional[Dict[str, Any]]:
        """Get a Devin session by Google Chat thread.

        Args:
            gchat_space: The Google Chat space ID.
            gchat_thread: The Google Chat thread ID.

        Returns:
            The session mapping document, or None if not found.
        """
        query = (
            self.collection
            .where("gchat_space", "==", gchat_space)
            .where("gchat_thread", "==", gchat_thread)
            .limit(1)
        )
        docs = query.stream()
        
        for doc in docs:
            return {"id": doc.id, **doc.to_dict()}
        
        return None

    def add_message(
        self, 
        mapping_id: str, 
        sender: str, 
        content: str
    ) -> None:
        """Add a message to a session mapping.

        Args:
            mapping_id: The document ID of the session mapping.
            sender: The sender of the message ("user" or "devin").
            content: The content of the message.
        """
        doc_ref = self.collection.document(mapping_id)
        doc_ref.update({
            "updated_at": firestore.SERVER_TIMESTAMP,
            "messages": firestore.ArrayUnion([{
                "sender": sender,
                "content": content,
                "timestamp": firestore.SERVER_TIMESTAMP
            }])
        })

    def get_user_sessions(self, user_id: str, limit: int = 10) -> List[Dict[str, Any]]:
        """Get recent sessions for a user.

        Args:
            user_id: The ID of the user.
            limit: The maximum number of sessions to return.

        Returns:
            A list of session mapping documents.
        """
        query = (
            self.collection
            .where("user_id", "==", user_id)
            .order_by("created_at", direction=firestore.Query.DESCENDING)
            .limit(limit)
        )
        
        return [{"id": doc.id, **doc.to_dict()} for doc in query.stream()]
