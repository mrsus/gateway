CREATE TABLE IF NOT EXISTS "libraries" (
  "id" SERIAL PRIMARY KEY,
  "api_id" INTEGER NOT NULL,
  "name" TEXT NOT NULL,
  "description" TEXT,
  "data" JSONB NOT NULL,
  UNIQUE ("api_id", "name"),
  FOREIGN KEY("api_id") REFERENCES "apis"("id") ON DELETE CASCADE
);
