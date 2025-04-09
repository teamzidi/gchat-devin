import os
import json
import logging
from typing import Dict, Any, Optional
from fastapi import FastAPI, Request, HTTPException, Depends
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field

from .devin_api import DevinAPIClient
from .firestore_client import FirestoreClient
from .secret_manager import SecretManagerClient

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="Google Chat Devin Bot")

PROJECT_ID = os.environ.get("GOOGLE_CLOUD_PROJECT")
DEVIN_API_KEY_SECRET = os.environ.get("DEVIN_API_KEY_SECRET", "devin-api-key")

secret_manager = None
devin_api = None
firestore = None

@app.on_event("startup")
async def startup_event():
    """Initialize clients on startup."""
    global secret_manager, devin_api, firestore
    
    try:
        secret_manager = SecretManagerClient(project_id=PROJECT_ID)
        
        devin_api_key = secret_manager.get_secret(DEVIN_API_KEY_SECRET)
        
        devin_api = DevinAPIClient(api_key=devin_api_key)
        
        firestore = FirestoreClient()
        
        logger.info("All clients initialized successfully")
    except Exception as e:
        logger.error(f"Error initializing clients: {e}")
        raise


class ChatEvent(BaseModel):
    """Model for Google Chat events."""
    type: str
    message: Optional[Dict[str, Any]] = None
    space: Dict[str, Any] = Field(...)
    user: Dict[str, Any] = Field(...)
    thread: Optional[Dict[str, Any]] = None


@app.post("/")
async def webhook(event: ChatEvent):
    """Handle Google Chat webhook events."""
    logger.info(f"Received event: {event.type}")
    
    if event.type != "MESSAGE":
        return JSONResponse(content={"text": "Event type not supported"})
    
    if not event.message:
        return JSONResponse(content={"text": "No message in event"})
    
    message_text = event.message.get("text", "")
    space_id = event.space.get("name", "")
    user_id = event.user.get("name", "")
    thread_id = event.thread.get("name") if event.thread else None
    
    if not thread_id:
        thread_id = event.message.get("name", "")
    
    is_mention = "@Devin" in message_text
    
    is_command = message_text.strip().startswith("/devin")
    
    if (is_mention or is_command) and "API" in message_text and ("usage" in message_text or "残り" in message_text):
        return await handle_api_usage_check(space_id, thread_id)
    
    if is_mention or is_command:
        clean_message = message_text.replace("@Devin", "").replace("/devin", "").strip()
        
        session_mapping = firestore.get_session_by_thread(space_id, thread_id)
        
        if session_mapping:
            return await handle_existing_session(session_mapping, clean_message, space_id, thread_id)
        else:
            return await handle_new_session(clean_message, space_id, thread_id, user_id)
    
    return JSONResponse(content={"text": ""})


async def handle_api_usage_check(space_id: str, thread_id: str) -> JSONResponse:
    """Handle API usage check request."""
    try:
        consumption_data = devin_api.get_enterprise_consumption()
        
        usage_text = "# Devin API Usage\n\n"
        
        if "remaining_credits" in consumption_data:
            usage_text += f"残りクレジット: {consumption_data['remaining_credits']}\n"
        
        if "total_credits" in consumption_data:
            usage_text += f"合計クレジット: {consumption_data['total_credits']}\n"
        
        if "usage_by_day" in consumption_data:
            usage_text += "\n## 日別使用量\n\n"
            for day_usage in consumption_data["usage_by_day"]:
                date = day_usage.get("date", "Unknown")
                credits = day_usage.get("credits", 0)
                usage_text += f"- {date}: {credits} credits\n"
        
        return JSONResponse(content={"text": usage_text})
    except Exception as e:
        logger.error(f"Error checking API usage: {e}")
        return JSONResponse(
            content={"text": "APIの使用状況を取得できませんでした。エラーが発生しました。"}
        )


async def handle_existing_session(
    session_mapping: Dict[str, Any], 
    message: str, 
    space_id: str, 
    thread_id: str
) -> JSONResponse:
    """Handle message for an existing Devin session."""
    try:
        devin_session_id = session_mapping["devin_session_id"]
        mapping_id = session_mapping["id"]
        
        devin_api.send_message(devin_session_id, message)
        
        firestore.add_message(mapping_id, "user", message)
        
        session_details = devin_api.get_session_details(devin_session_id)
        session_url = session_details.get("url", "")
        
        response_text = (
            f"メッセージを Devin に送信しました。Devin は応答を処理しています。\n\n"
            f"セッション URL: {session_url}"
        )
        
        return JSONResponse(content={"text": response_text})
    except Exception as e:
        logger.error(f"Error handling existing session: {e}")
        return JSONResponse(
            content={"text": "Devin にメッセージを送信できませんでした。エラーが発生しました。"}
        )


async def handle_new_session(
    prompt: str, space_id: str, thread_id: str, user_id: str
) -> JSONResponse:
    """Handle creation of a new Devin session."""
    try:
        session = devin_api.create_session(prompt)
        
        devin_session_id = session.get("session_id")
        session_url = session.get("url", "")
        
        if not devin_session_id:
            raise ValueError("No session ID returned from Devin API")
        
        firestore.create_session_mapping(
            gchat_space=space_id,
            gchat_thread=thread_id,
            devin_session_id=devin_session_id,
            user_id=user_id,
            prompt=prompt
        )
        
        response_text = (
            f"新しい Devin セッションを作成しました。\n\n"
            f"プロンプト: {prompt}\n\n"
            f"セッション URL: {session_url}\n\n"
            f"このスレッドで会話を続けることができます。"
        )
        
        return JSONResponse(content={"text": response_text})
    except Exception as e:
        logger.error(f"Error creating new session: {e}")
        return JSONResponse(
            content={"text": "Devin セッションを作成できませんでした。エラーが発生しました。"}
        )


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {"status": "healthy"}
