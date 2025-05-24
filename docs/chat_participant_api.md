# AssistDeck Chat Participant API Documentation

## Overview
The Chat Participant API allows users to share chat sessions with other users. This enables collaborative conversations where multiple users can connect to the same chat session and exchange messages in real-time.

## Authentication
All API endpoints require authentication using a valid JWT token in the `Authorization` header with the format: `Bearer <token>`.

## Models

### ChatParticipant
The ChatParticipant model represents a user who has been given access to a chat session:

```json
{
  "id": "uuid",
  "session_id": "uuid",
  "user_id": "uuid",
  "created_at": "timestamp"
}
```

## Endpoints

### Add a Participant

Adds a user as a participant to a chat session. Only the session owner can add participants.

**URL:** `/api/chat/sessions/:id/participants`
**Method:** `POST`
**URL Params:** 
- `id`: The UUID of the chat session

**Request Body:**
```json
{
  "user_id": "uuid-of-user-to-add"
}
```

**Success Response:**
- **Code:** 201 CREATED
- **Content:** 
```json
{
  "id": "uuid",
  "session_id": "uuid",
  "user_id": "uuid",
  "created_at": "timestamp"
}
```

**Error Responses:**
- **Code:** 400 BAD REQUEST - Invalid input or missing required fields
- **Code:** 401 UNAUTHORIZED - Missing or invalid token
- **Code:** 403 FORBIDDEN - Only the session owner can add participants
- **Code:** 404 NOT FOUND - Session not found
- **Code:** 409 CONFLICT - User is already a participant

### List Session Participants

Returns a list of all participants for a chat session. Both the session owner and participants can access this endpoint.

**URL:** `/api/chat/sessions/:id/participants`
**Method:** `GET`
**URL Params:** 
- `id`: The UUID of the chat session

**Success Response:**
- **Code:** 200 OK
- **Content:** Array of participants including the owner
```json
[
  {
    "id": "uuid",
    "user_id": "uuid",
    "email": "user@example.com",
    "name": "User Name",
    "is_owner": true,
    "created_at": "timestamp"
  },
  {
    "id": "uuid",
    "user_id": "uuid",
    "email": "participant@example.com",
    "name": "Participant Name",
    "is_owner": false,
    "created_at": "timestamp"
  }
]
```

**Error Responses:**
- **Code:** 400 BAD REQUEST - Invalid session ID
- **Code:** 401 UNAUTHORIZED - Missing or invalid token
- **Code:** 403 FORBIDDEN - User doesn't have access to this chat session
- **Code:** 404 NOT FOUND - Session not found

### Remove a Participant

Removes a participant from a chat session. Only the session owner can remove participants.

**URL:** `/api/chat/sessions/:id/participants/:userId`
**Method:** `DELETE`
**URL Params:** 
- `id`: The UUID of the chat session
- `userId`: The UUID of the user to remove

**Success Response:**
- **Code:** 200 OK
- **Content:** 
```json
{
  "message": "participant removed"
}
```

**Error Responses:**
- **Code:** 400 BAD REQUEST - Invalid session ID or participant ID
- **Code:** 401 UNAUTHORIZED - Missing or invalid token
- **Code:** 403 FORBIDDEN - Only the session owner can remove participants
- **Code:** 404 NOT FOUND - Session or participant not found

## WebSocket Connection

Both the session owner and participants can connect to the chat session's WebSocket to receive real-time updates.

**WebSocket URL:** `ws://[server]/ws?session_id=[session_id]&token=[jwt_token]`

The WebSocket connection handler verifies that the user is either the owner of the session or has been added as a participant before allowing the connection.

## Session Listing

When listing chat sessions for a user, the API includes both sessions owned by the user and sessions where the user has been added as a participant.

**URL:** `/api/chat/sessions`
**Method:** `GET`

This query combines owned sessions and sessions where the user is a participant into a single list.

## Usage Examples

### Example: Adding a participant to a chat session

```javascript
// Add participant
const response = await fetch(`http://localhost:8080/api/chat/sessions/${sessionId}/participants`, {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${ownerToken}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    user_id: participantUserId
  })
});

// Connect to WebSocket as owner
const ownerSocket = new WebSocket(`ws://localhost:8080/ws?session_id=${sessionId}&token=${ownerToken}`);

// Connect to WebSocket as participant
const participantSocket = new WebSocket(`ws://localhost:8080/ws?session_id=${sessionId}&token=${participantToken}`);
```
