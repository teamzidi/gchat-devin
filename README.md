# Google Chat Devin Bot

Google Chat integration for Devin AI. This bot allows you to interact with Devin directly from Google Chat.

## Features

- Start Devin sessions with `@Devin` mentions or `/devin` commands
- Continue conversations with Devin in threads
- Check API usage with `@Devin API usage` or `/devin API usage`

## Architecture

- **Backend**: Go application deployed to Google Cloud Run
- **Database**: Google Cloud Firestore for storing session mappings
- **Secrets**: Google Cloud Secret Manager for storing API keys

## Setup

### Prerequisites

- Google Cloud Project with the following APIs enabled:
  - Cloud Run
  - Firestore
  - Secret Manager
- Devin API key
- Google Workspace with Google Chat enabled

### Deployment Steps

1. Clone this repository:
   ```
   git clone https://github.com/teamzidi/gchat-devin.git
   cd gchat-devin
   ```

2. Store your Devin API key in Secret Manager:
   ```
   gcloud secrets create devin-api-key --replication-policy="automatic"
   echo -n "your-devin-api-key" | gcloud secrets versions add devin-api-key --data-file=-
   ```

3. Deploy to Cloud Run:
   ```
   gcloud builds submit --tag gcr.io/YOUR_PROJECT_ID/gchat-devin
   gcloud run deploy gchat-devin \
     --image gcr.io/YOUR_PROJECT_ID/gchat-devin \
     --platform managed \
     --region asia-northeast1 \
     --allow-unauthenticated \
     --set-env-vars="GOOGLE_CLOUD_PROJECT=YOUR_PROJECT_ID,DEVIN_API_KEY_SECRET=devin-api-key"
   ```

4. Configure Google Chat webhook:
   - Go to Google Chat API in the Google Cloud Console
   - Create a new Chat app
   - Set up the app as a bot
   - Configure the bot to use the Cloud Run URL as the webhook endpoint
   - Add the necessary scopes and permissions

## Local Development

1. Install Go dependencies:
   ```
   go mod download
   ```

2. Set up environment variables:
   ```
   export GOOGLE_CLOUD_PROJECT=your-project-id
   export DEVIN_API_KEY_SECRET=devin-api-key
   ```

3. Run the application:
   ```
   go run cmd/server/main.go
   ```

## Testing

You can test the webhook locally using tools like ngrok to expose your local server to the internet.

## License

See the [LICENSE](LICENSE) file for details.
