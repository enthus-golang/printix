# Printix Cloud Print API Documentation

*This document is a comprehensive transcription of the official Printix Cloud Print API documentation from [https://printix.github.io/](https://printix.github.io/)*

## Changelog

- **2025-04-24:** It is now possible to query normal users in addition to guest users by using List Users endpoint.
- **2025-04-16:** Added endpoints for site management, network management, and SNMP configuration management.
- **2024-12-17:** Removed support for refresh tokens when using the client_credential OAuth2 flow.
- **2024-11-14:** Added endpoints for group management.
- **2024-10-05:** Expose serial number on printers.
- **2024-04-06:** Expose model, vendor and location fields on printers.
- **2023-07-12:** Added endpoints for guest user administration.
- **2023-05-24:** Added the `/changeOwner` endpoint.
- **2021-07-30:** Exposed the printer status (online/offline).
- **2021-03-26:** Better documented the use for the PDL parameter.
- **2021-03-22:** Added an option to specify if a job should be scaled.
- **2021-03-01:** Added an alternative version of the `/submit` endpoint (v1.1).

## Introduction

The Printix Cloud Print API is intended for use by applications that wish to push print jobs in PDF or native format into a Printix queue.

Creating and printing a document is a multistage process:
1.  Fetching a list of printers. (See **List Printers**)
2.  Selecting a printer. (See **Get printer properties**)
3.  Creating a job for that printer. (See **Submit a job**)
4.  Making a PUT request to the cloud storage links returned in the previous step's response, including the document to print in the request body. (See **Upload a file**)
5.  Using the `uploadCompleted` link from step 3 to inform Printix that the document is ready to be printed. (See **Complete Upload**).

## API Conventions

- The API uses the HAL document format and is discoverable from the API entry-point (`https://api.printix.net/cloudprint`).
- Consumers of the API are encouraged to not manually construct URLs, but instead use the links returned in API responses.
- Resource lists (jobs, printers) are returned in a paginated container format.

## Authentication

Authentication is handled via the OAuth 2.0 Client Credentials flow. You can obtain a client ID and secret from the Printix Administrator dashboard.

To get an access token, send a POST request to the Printix token endpoint:

```bash
curl 'https://auth.printix.net/oauth/token' -X POST \
  -H 'Content-Type: application/x-www-form-urlencoded'\
  -d 'grant_type=client_credentials&client_id={clientId}&client_secret={clientSecret}'
```

A successful response will look like this:
```json
{
  "access_token": "5a8c4ec4-ff70-4f4c-b088-29a8cae38062",
  "expires_in": 3599
}
```
Each access token is valid for one hour.

## Webhooks

Webhooks allow you to subscribe to events in Printix. You can register a URL and select event types. Each webhook has two shared secrets used to compute an HMAC-SHA512 signature, which is included in the `X-Printix-Signature` header. This allows for key rotation without downtime.

To prevent replay attacks, you should only process events with a recent timestamp (within ±15 minutes), sent in the `X-Printix-Timestamp` header.

### User Creation Event Example
```bash
curl –X POST https://{webhook-endpoint} \
 -H "X-Printix-Signature-1: 4d29..." \
 -H "X-Printix-Signature-2: 3646..." \
 -H "X-Printix-Request-ID: 2196..." \
 -H "Content-Type: application/json" \
 --data '{
   "emitted": 1718093846.488,
   "events": [
     {
       "name": "RESOURCE.TENANT_USER.CREATE",
       "href": "https://api.printix.net/cloudprint/tenants/{tenant}/users/{user}",
       "time": 1718093846.488
     }
   ]
 }'
```

## Rate Limiting

The API uses a token bucket algorithm, limited to **100 requests per minute per user**.
- On a successful request, the `X-Rate-Limit-Remaining` header indicates the remaining tokens.
- If throttled (HTTP 429), the `X-Rate-Limit-Retry-After-Seconds` header indicates how long to wait.

---

## API Endpoints

### Root

#### **`GET /cloudprint`**
The API entry point. It returns a list of links to accessible tenants.

### Job Management

#### **`POST /cloudprint/tenants/{tenantId}/printers/{printerId}/jobs`** (Submit a job)
Creates a new Printix job.

**Request parameters**
| Parameter | Description |
|---|---|
| `title` | Title of the print job. |
| `user` | Name of the user that is printing. This parameter is used for integration with third-party print solutions. |
| `PDL` | Optional. Printer Document Language to use for the document to be printed, if it is not PDF. (e.g., `PCL5`, `POSTSCRIPT`, `XPS`) |

**Response Body Fields**
| Path | Type | Description |
|---|---|---|
| `job.id` | String | ID of the generated print job. |
| `job.createTime` | Number | Unix timestamp for when the job was created. |
| `job.updateTime` | Number | Unix timestamp of when the job was last updated. |
| `job.status` | String | Current state of the job. |
| `job.ownerId` | String | ID of the Printix user who is the owner of the job. |
| `job.contentType` | String | Content type of the uploaded document. |
| `job.title` | String | Title of the print job. |
| `_links.uploadCompleted` | Object | Link to be used once a job has completed uploading. |
| `uploadLinks` | Array | Document upload links. |
| `uploadLinks..url` | String | The actual URL to upload the document data (using HTTP PUT). |
| `uploadLinks..headers` | Object | Any additional headers to add to the HTTP PUT request. |

#### **New Submit Method (v1.1)**
To use this, add a `version: 1.1` header and send a JSON body with print properties.

**Request body fields**
| Parameter | Description |
|---|---|
| `color` | Sets whether the document should be printed in color. Possible values are `true` or `false`. |
| `duplex` | Sets what kind of duplex to use. Possible values are `NONE`, `SHORT_EDGE` or `LONG_EDGE`. |
| `page_orientation` | Sets the page orientation. Possible values are `PORTRAIT`, `LANDSCAPE` or `AUTO`. |
| `copies` | A positive integer, representing the number of copies to print. |
| `media_size` | Sets the page dimensions. (e.g., `A4`, `LETTER`). |
| `scaling` | Determines how the job should be scaled. Possible values are: `NOSCALE`, `SHRINK`, `FIT`. |

#### **Upload a file**
After submitting the job, upload the file to the URL provided in `uploadLinks` using a `PUT` request.

#### **Complete Upload**
Call the `uploadCompleted` link from the submit response to notify Printix the upload is done.

#### **`GET /cloudprint/tenants/{tenantId}/jobs`** (Retrieve Jobs)
Retrieves a list of jobs for the tenant.

#### **`GET /cloudprint/tenants/{tenantId}/jobs/{jobId}`** (Retrieve a single Job)
Retrieves details for a specific job.

#### **`DELETE /cloudprint/tenants/{tenantId}/jobs/{jobId}`** (Delete a single Job)
Deletes a job.

### Printer Management

#### **`GET /cloudprint/tenants/{tenantId}/printers`** (List Printers)
Retrieves a list of printers.

#### **`GET /cloudprint/tenants/{tenantId}/printers/{printerId}`** (Get printer properties)
Retrieves all properties and capabilities for a specific printer.

### User Management

#### **`POST /cloudprint/tenants/{tenantId}/users`** (Create User)
Creates a new guest user.

**Request body fields**
| Parameter | Description |
|---|---|
| `email` | Email address of the user. |
| `fullName` | Full name of the user. |
| `role` | Role of the user. Currently, only the `GUEST_USER` role is supported. |
| `pin` | An optional, 4-digit PIN code for the user. |
| `password` | An optional password for the user. |

### And many more endpoints for Groups, Cards, Sites, Networks, and SNMP Configurations with full CRUD operations...

---

## Code Examples

### Root Example

**Request**
```bash
curl 'https://api.printix.net/cloudprint' -i -X GET \
    -H 'Authorization: Bearer {your-token}'
```

**Response**
```json
HTTP/1.1 200 OK
Content-Type: application/json;charset=UTF-8

{
    "_links": {
        "self": {
            "href": "https://api.printix.net/cloudprint"
        },
        "tenants": [
            {
                "href": "https://api.printix.net/cloudprint/tenants/b54aed12-c905-4dd1-a69f-5b34aec43533"
            }
        ]
    },
    "success": true,
    "message": "OK"
}
```

### Submit Example (v1.0)

**Request**
```bash
curl 'https://api.printix.net/cloudprint/tenants/b54aed12-c905-4dd1-a69f-5b34aec43533/printers/581cd264-ccda-4407-b0c5-fb87bd42e95f/jobs?title=My+Document' -i -X POST \
    -H 'Authorization: Bearer {your-token}'
```

**Response**
```json
HTTP/1.1 200 OK
Content-Type: application/json;charset=UTF-8

{
    "success": true,
    "job": {
        "id": "c5d97124-4e31-4a83-887b-b52b6d53249e",
        "createTime": 1600344674,
        "updateTime": 1600344674,
        "status": "Created",
        "ownerId": "f3906903-d989-4065-9148-c588192e63e1",
        "contentType": "application/pdf",
        "title": "My Document"
    },
    "uploadLinks": [
        {
            "url": "https://printixjobs.blob.core.windows.net/9aab9505-c84f-4dba-814f-d710b7c6d089/printix-jobs-file-upload/c5d97124-4e31-4a83-887b-b52b6d53249e?sig=3FlDWWBnr7704VgZjjApglkioBcUkS9F%2FZXJAVbBmAY%3D&st=2020-09-17T12%3A11%3A14Z&se=2020-09-17T13%3A26%3A14Z&sv=2017-04-17&sp=cw&sr=b",
            "headers": {
                "x-ms-blob-type": "BlockBlob"
            },
            "type": "Azure"
        }
    ],
    "_links": {
        "self": {
            "href": "https://api.printix.net/cloudprint/tenants/b54aed12-c905-4dd1-a69f-5b34aec43533/jobs/c5d97124-4e31-4a83-887b-b52b6d53249e"
        },
        "uploadCompleted": {
            "href": "https://api.printix.net/cloudprint/completeUpload?jobId=c5d97124-4e31-4a83-887b-b52b6d53249e"
        }
    }
}
```

### Submit Example (v1.1)

**Request**
```bash
curl 'https://api.printix.net/cloudprint/tenants/b54aed12-c905-4dd1-a69f-5b34aec43533/printers/581cd264-ccda-4407-b0c5-fb87bd42e95f/jobs?title=My+Document' -i -X POST \
    -H 'Authorization: Bearer {your-token}' \
    -H 'version: 1.1' \
    -H 'Content-Type: application/json' \
    -d '{
          "color": false,
          "duplex": "LONG_EDGE",
          "page_orientation": "PORTRAIT",
          "copies": 1,
          "media_size": "A4",
          "scaling": "FIT"
        }'
```

### Complete Upload Example

**Request**
```bash
curl 'https://api.printix.net/cloudprint/completeUpload?jobId=c5d97124-4e31-4a83-887b-b52b6d53249e' -i -X POST \
    -H 'Authorization: Bearer {your-token}'
```

**Response**
```json
HTTP/1.1 200 OK
Content-Type: application/json;charset=UTF-8

{
    "success": true,
    "job": {
        "id": "c5d97124-4e31-4a83-887b-b52b6d53249e",
        "createTime": 1600344674,
        "updateTime": 1600344674,
        "status": "Processing",
        "ownerId": "f3906903-d989-4065-9148-c588192e63e1",
        "contentType": "application/pdf",
        "title": "My Document"
    }
}