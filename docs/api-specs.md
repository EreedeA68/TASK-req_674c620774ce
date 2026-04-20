# MeridianMart Offline Commerce — REST API Specification

## Global Headers

All requests must include the following headers for replay prevention:

| Header | Description |
|--------|-------------|
| `X-Timestamp` | Unix epoch seconds (request rejected if outside ±5-minute window) |
| `X-Nonce` | Unique random UUID per request (rejected if nonce was used within the last 10 minutes) |
| `Authorization` | `Bearer <jwt>` (required on all authenticated endpoints) |

All responses use `Content-Type: application/json` unless noted otherwise.

---

## Auth Endpoints

### POST /api/auth/login

**Role required:** None (public)

**Description:** Authenticates a shopper, staff member, or admin with username and password. Returns a JWT on success. Locks the account for 15 minutes after 5 consecutive failures.

**Request headers:** `X-Timestamp`, `X-Nonce`

**Request body:**
```json
{
  "username": "string",
  "password": "string"
}
```

**Success response — HTTP 200:**
```json
{
  "token": "string (JWT)",
  "userId": "long",
  "role": "SHOPPER | STAFF | ADMIN | READONLY",
  "expiresAt": "ISO-8601 datetime"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 400 | Missing or blank username or password |
| 401 | Invalid credentials |
| 403 | Account locked due to brute-force protection |
| 400 | Invalid or expired X-Timestamp / duplicate X-Nonce |
| 429 | Rate limit exceeded |

---

### POST /api/auth/logout

**Role required:** Any authenticated user

**Description:** Revokes the current session token.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Request body:** None

**Success response — HTTP 200:**
```json
{
  "message": "Logged out successfully"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 401 | Missing or invalid JWT |
| 400 | Invalid X-Timestamp or duplicate X-Nonce |

---

### GET /api/auth/me

**Role required:** Any authenticated user

**Description:** Returns the currently authenticated user's profile.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Success response — HTTP 200:**
```json
{
  "userId": "long",
  "username": "string",
  "role": "SHOPPER | STAFF | ADMIN | READONLY",
  "lastLoginAt": "ISO-8601 datetime"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 401 | Missing or invalid JWT |

---

## Shopper Endpoints

### GET /api/products

**Role required:** SHOPPER, STAFF, ADMIN, READONLY

**Description:** Returns paginated list of active products in the catalog.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Query parameters:**
- `page` (int, default 0)
- `size` (int, default 20)
- `category` (long, optional)
- `search` (string, optional)

**Success response — HTTP 200:**
```json
{
  "content": [
    {
      "productId": "long",
      "name": "string",
      "description": "string",
      "categoryId": "long",
      "categoryName": "string",
      "price": "decimal",
      "onHandQty": "int",
      "stockWarning": "boolean (true when onHandQty < 2)"
    }
  ],
  "page": "int",
  "size": "int",
  "totalElements": "long",
  "totalPages": "int"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 401 | Missing or invalid JWT |
| 429 | Rate limit exceeded |

---

### GET /api/products/{id}

**Role required:** SHOPPER, STAFF, ADMIN, READONLY

**Description:** Returns full product details including stock warning and Top-10 recommendations for this product's page.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Path parameters:**
- `id` (long) — product ID

**Success response — HTTP 200:**
```json
{
  "productId": "long",
  "name": "string",
  "description": "string",
  "categoryId": "long",
  "categoryName": "string",
  "price": "decimal",
  "onHandQty": "int",
  "stockWarning": "boolean (true when onHandQty < 2)",
  "averageRating": "decimal (1.0–5.0, null if no ratings)",
  "recommendations": [
    {
      "productId": "long",
      "name": "string",
      "price": "decimal"
    }
  ]
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 401 | Missing or invalid JWT |
| 404 | Product not found or inactive |
| 429 | Rate limit exceeded |

---

### POST /api/cart

**Role required:** SHOPPER

**Description:** Adds a product to the shopper's cart or updates quantity if already present.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Request body:**
```json
{
  "productId": "long",
  "quantity": "int (min 1)"
}
```

**Success response — HTTP 200:**
```json
{
  "message": "Added to cart",
  "cartItemId": "long",
  "productId": "long",
  "quantity": "int",
  "stockWarning": "boolean (true when onHandQty < 2)"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 400 | Invalid productId or quantity < 1 |
| 401 | Missing or invalid JWT |
| 403 | Caller role is not SHOPPER |
| 404 | Product not found or inactive |
| 409 | Requested quantity exceeds on-hand stock |
| 429 | Rate limit exceeded |

---

### GET /api/cart

**Role required:** SHOPPER

**Description:** Returns all items in the shopper's current cart.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Success response — HTTP 200:**
```json
{
  "items": [
    {
      "cartItemId": "long",
      "productId": "long",
      "name": "string",
      "price": "decimal",
      "quantity": "int",
      "lineTotal": "decimal",
      "stockWarning": "boolean"
    }
  ],
  "cartTotal": "decimal"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 401 | Missing or invalid JWT |
| 403 | Caller role is not SHOPPER |

---

### DELETE /api/cart/{itemId}

**Role required:** SHOPPER

**Description:** Removes a specific item from the shopper's cart.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Path parameters:**
- `itemId` (long) — cart item ID

**Success response — HTTP 200:**
```json
{
  "message": "Item removed from cart"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 401 | Missing or invalid JWT |
| 403 | Caller role is not SHOPPER, or item does not belong to caller |
| 404 | Cart item not found |

---

### POST /api/favorites

**Role required:** SHOPPER

**Description:** Adds a product to the shopper's favorites list.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Request body:**
```json
{
  "productId": "long"
}
```

**Success response — HTTP 201:**
```json
{
  "favoriteId": "long",
  "productId": "long",
  "addedAt": "ISO-8601 datetime"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 400 | Invalid productId |
| 401 | Missing or invalid JWT |
| 403 | Caller role is not SHOPPER |
| 404 | Product not found or inactive |
| 409 | Product already in favorites |

---

### GET /api/favorites

**Role required:** SHOPPER

**Description:** Returns all products in the shopper's favorites list.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Success response — HTTP 200:**
```json
{
  "favorites": [
    {
      "favoriteId": "long",
      "productId": "long",
      "name": "string",
      "price": "decimal",
      "addedAt": "ISO-8601 datetime"
    }
  ]
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 401 | Missing or invalid JWT |
| 403 | Caller role is not SHOPPER |

---

### DELETE /api/favorites/{id}

**Role required:** SHOPPER

**Description:** Removes a product from the shopper's favorites list.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Path parameters:**
- `id` (long) — favorite record ID

**Success response — HTTP 200:**
```json
{
  "message": "Removed from favorites"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 401 | Missing or invalid JWT |
| 403 | Caller role is not SHOPPER, or record does not belong to caller |
| 404 | Favorite record not found |

---

### POST /api/orders

**Role required:** SHOPPER

**Description:** Creates an order from the current cart contents and initiates the checkout flow. Generates a receipt number and records the transaction timestamp. Returns a purchase confirmation with receipt details.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Request body:**
```json
{
  "idempotencyKey": "string (UUID)",
  "paymentMethod": "WECHAT_PAY_OFFLINE",
  "posConfirmationRef": "string"
}
```

**Success response — HTTP 201:**
```json
{
  "orderId": "long",
  "receiptNumber": "string",
  "transactionTimestamp": "string (12-hour format, e.g. '04/18/2026 02:35:10 PM')",
  "status": "CONFIRMED",
  "items": [
    {
      "productId": "long",
      "name": "string",
      "quantity": "int",
      "unitPrice": "decimal",
      "lineTotal": "decimal"
    }
  ],
  "totalAmount": "decimal",
  "message": "Purchase confirmed. Your receipt number is {receiptNumber}."
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 400 | Empty cart, missing idempotencyKey, invalid posConfirmationRef |
| 401 | Missing or invalid JWT |
| 403 | Caller role is not SHOPPER |
| 409 | Duplicate idempotencyKey (returns existing order) |
| 409 | Distributed lock could not be acquired (concurrent payment attempt) |
| 422 | One or more cart items out of stock |
| 429 | Rate limit exceeded |

---

### GET /api/orders

**Role required:** SHOPPER

**Description:** Returns the shopper's order history.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Query parameters:**
- `page` (int, default 0)
- `size` (int, default 20)

**Success response — HTTP 200:**
```json
{
  "content": [
    {
      "orderId": "long",
      "receiptNumber": "string",
      "status": "PENDING | CONFIRMED | READY_FOR_PICKUP | COMPLETED | REFUNDED | CANCELLED",
      "totalAmount": "decimal",
      "createdAt": "ISO-8601 datetime",
      "transactionTimestamp": "string (12-hour format)"
    }
  ],
  "page": "int",
  "totalElements": "long"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 401 | Missing or invalid JWT |
| 403 | Caller role is not SHOPPER |

---

### POST /api/ratings

**Role required:** SHOPPER

**Description:** Submits a 1–5 star rating for a purchased product. The order must be in COMPLETED status.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Request body:**
```json
{
  "productId": "long",
  "orderId": "long",
  "score": "int (1–5)"
}
```

**Success response — HTTP 201:**
```json
{
  "ratingId": "long",
  "productId": "long",
  "score": "int",
  "createdAt": "ISO-8601 datetime"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 400 | Score out of range (not 1–5), missing fields |
| 401 | Missing or invalid JWT |
| 403 | Caller role is not SHOPPER, or order does not belong to caller |
| 404 | Product or order not found |
| 409 | Rating already submitted for this product and order |
| 422 | Order is not in COMPLETED status |

---

### GET /api/notifications

**Role required:** SHOPPER

**Description:** Returns in-app notifications for the shopper. Supports filtering by read/unread status.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Query parameters:**
- `unreadOnly` (boolean, default false)

**Success response — HTTP 200:**
```json
{
  "notifications": [
    {
      "notificationId": "long",
      "type": "ORDER_STATUS | PICKUP_READY",
      "message": "string",
      "orderId": "long (nullable)",
      "read": "boolean",
      "createdAt": "ISO-8601 datetime"
    }
  ],
  "unreadCount": "int"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 401 | Missing or invalid JWT |
| 403 | Caller role is not SHOPPER |

---

### POST /api/behavior

**Role required:** SHOPPER

**Description:** Records a shopper behavior event for the recommendation engine (view, favorite, add-to-cart). Purchase and rating events are recorded automatically by other endpoints.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Request body:**
```json
{
  "productId": "long",
  "eventType": "VIEW | FAVORITE | ADD_TO_CART"
}
```

**Success response — HTTP 201:**
```json
{
  "eventId": "long",
  "recorded": true
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 400 | Invalid eventType or productId |
| 401 | Missing or invalid JWT |
| 403 | Caller role is not SHOPPER |
| 404 | Product not found |

---

### GET /api/recommendations

**Role required:** SHOPPER

**Description:** Returns the shopper's Top-10 personalized product recommendations. Serves cached results if not expired (60-minute TTL). Falls back to category popularity and new arrivals for cold-start shoppers.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Success response — HTTP 200:**
```json
{
  "recommendations": [
    {
      "productId": "long",
      "name": "string",
      "price": "decimal",
      "score": "float",
      "rank": "int (1–10)",
      "coldStart": "boolean"
    }
  ],
  "generatedAt": "ISO-8601 datetime",
  "expiresAt": "ISO-8601 datetime"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 401 | Missing or invalid JWT |
| 403 | Caller role is not SHOPPER |

---

## Staff Endpoints

### GET /api/transactions/{receiptNumber}

**Role required:** STAFF, ADMIN

**Description:** Looks up an order and its payment details by receipt number.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Path parameters:**
- `receiptNumber` (string) — unique receipt number from the order confirmation screen

**Success response — HTTP 200:**
```json
{
  "orderId": "long",
  "receiptNumber": "string",
  "userId": "long",
  "status": "string",
  "totalAmount": "decimal",
  "transactionTimestamp": "string (12-hour format)",
  "items": [
    {
      "productId": "long",
      "name": "string",
      "quantity": "int",
      "unitPrice": "decimal",
      "lineTotal": "decimal"
    }
  ],
  "payment": {
    "paymentId": "long",
    "method": "WECHAT_PAY_OFFLINE",
    "status": "string",
    "amount": "decimal",
    "posConfirmationRef": "string (masked to last 4 chars)",
    "settledAt": "ISO-8601 datetime (nullable)"
  }
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 401 | Missing or invalid JWT |
| 403 | Caller role is not STAFF or ADMIN |
| 404 | No order found for the given receipt number |

---

### POST /api/refunds

**Role required:** STAFF, ADMIN

**Description:** Initiates a refund for an order. Records an idempotent refund request tied to the original payment.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Request body:**
```json
{
  "orderId": "long",
  "amount": "decimal",
  "reason": "string",
  "idempotencyKey": "string (UUID)"
}
```

**Success response — HTTP 201:**
```json
{
  "refundId": "long",
  "orderId": "long",
  "amount": "decimal",
  "status": "PENDING | COMPLETED",
  "createdAt": "ISO-8601 datetime"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 400 | Missing fields, amount exceeds original payment, invalid orderId |
| 401 | Missing or invalid JWT |
| 403 | Caller role is not STAFF or ADMIN |
| 404 | Order not found |
| 409 | Duplicate idempotencyKey (returns existing refund) |
| 422 | Order is not in a refundable status (must be CONFIRMED or COMPLETED) |

---

### PUT /api/orders/{id}/ready-for-pickup

**Role required:** STAFF, ADMIN

**Description:** Marks an order as ready for pickup. Triggers a pickup-ready notification to the shopper (subject to the 5 notifications/day cap).

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Path parameters:**
- `id` (long) — order ID

**Request body:** None

**Success response — HTTP 200:**
```json
{
  "orderId": "long",
  "status": "READY_FOR_PICKUP",
  "notificationSent": "boolean (false if daily cap reached)"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 401 | Missing or invalid JWT |
| 403 | Caller role is not STAFF or ADMIN |
| 404 | Order not found |
| 422 | Order is not in CONFIRMED status |

---

## Admin Endpoints

### GET /api/feature-flags

**Role required:** ADMIN

**Description:** Returns all configured feature flags and their current enabled state.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Success response — HTTP 200:**
```json
{
  "flags": [
    {
      "flagId": "long",
      "name": "string",
      "enabled": "boolean",
      "description": "string",
      "updatedBy": "long (userId)",
      "updatedAt": "ISO-8601 datetime"
    }
  ]
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 401 | Missing or invalid JWT |
| 403 | Caller role is not ADMIN |

---

### PUT /api/feature-flags/{id}

**Role required:** ADMIN

**Description:** Enables or disables a feature flag. Change is recorded in `audit_logs`.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Path parameters:**
- `id` (long) — feature flag ID

**Request body:**
```json
{
  "enabled": "boolean"
}
```

**Success response — HTTP 200:**
```json
{
  "flagId": "long",
  "name": "string",
  "enabled": "boolean",
  "updatedAt": "ISO-8601 datetime"
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 400 | Missing or invalid `enabled` field |
| 401 | Missing or invalid JWT |
| 403 | Caller role is not ADMIN |
| 404 | Feature flag not found |

---

### GET /api/compliance-reports

**Role required:** ADMIN

**Description:** Returns a compliance and reconciliation report summarizing payments, refunds, and net settlement for a date range.

**Request headers:** `Authorization`, `X-Timestamp`, `X-Nonce`

**Query parameters:**
- `from` (ISO-8601 date, required)
- `to` (ISO-8601 date, required)
- `type` (string, optional: `PAYMENTS` | `REFUNDS` | `RECONCILIATION` | `AUDIT` — defaults to all)

**Success response — HTTP 200:**
```json
{
  "reportType": "string",
  "from": "ISO-8601 date",
  "to": "ISO-8601 date",
  "generatedAt": "ISO-8601 datetime",
  "summary": {
    "totalOrders": "int",
    "totalPayments": "decimal",
    "totalRefunds": "decimal",
    "netSettlement": "decimal",
    "failedPayments": "int"
  },
  "entries": [
    {
      "date": "ISO-8601 date",
      "orderId": "long",
      "receiptNumber": "string",
      "paymentStatus": "string",
      "amount": "decimal",
      "refundAmount": "decimal (0 if none)",
      "posConfirmationRef": "string (masked)"
    }
  ]
}
```

**Error cases:**
| Status | Condition |
|--------|-----------|
| 400 | Missing or invalid date parameters, `from` after `to` |
| 401 | Missing or invalid JWT |
| 403 | Caller role is not ADMIN |
