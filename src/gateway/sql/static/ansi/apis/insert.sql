INSERT INTO apis (account_id, name, description,
  cors_allow_origin, cors_allow_headers, cors_allow_credentials,
  cors_request_headers, cors_max_age, enable_swagger, created_at)
VALUES (
  ?, ?, ?,
  ?, ?, ?,
  ?, ?, ?, CURRENT_TIMESTAMP)
