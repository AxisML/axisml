DROP INDEX IF EXISTS mlservices_admission_order;
DROP INDEX IF EXISTS mlservices_sync_pending;
CREATE INDEX mlservices_sync_pending
    ON mlservices(id)
    WHERE generation <> observed_generation AND deleted_at IS NULL;
ALTER TABLE mlservices ALTER COLUMN phase SET DEFAULT 'Creating';
ALTER TABLE mlservices DROP COLUMN IF EXISTS dispatched_replicas;
ALTER TABLE mlservices DROP COLUMN IF EXISTS admitted_replicas;
