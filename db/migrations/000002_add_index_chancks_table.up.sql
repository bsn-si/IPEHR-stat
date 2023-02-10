CREATE TABLE IF NOT EXISTS "tree_indexe_chunks" (
    "key" TEXT NOT NULL,
    "created_at" INTEGER DEFAULT CURRENT_TIMESTAMP,
    "group_id" TEXT NOT NULL,
    "data_id" TEXT NOT NULL,
    "ehr_id" TEXT NOT NULL,
    "data" BLOB NOT NULL,
    "hash" TEXT NOT NULL,
    PRIMARY KEY("key")
);