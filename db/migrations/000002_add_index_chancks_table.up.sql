CREATE TABLE IF NOT EXISTS "bk_indexes" (
    "key" TEXT NOT NULL,
    "created_at" INTEGER DEFAULT CURRENT_TIMESTAMP,
    "data" BLOB,
    "hash" TEXT,
    PRIMARY KEY("key")
);