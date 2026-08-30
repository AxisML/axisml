-- Durable, incremental MLService admission. Desired replicas remain in spec;
-- admitted_replicas is the capacity/quota reservation and
-- dispatched_replicas is the last replica vector accepted by the runtime.
ALTER TABLE mlservices
    ADD COLUMN admitted_replicas jsonb NOT NULL DEFAULT '[]',
    ADD COLUMN dispatched_replicas jsonb NOT NULL DEFAULT '[]';

-- Existing services have already been submitted through the legacy path, so
-- preserve their current desired replica vector as both admitted and
-- dispatched. New rows are inserted explicitly with zeroed vectors.
UPDATE mlservices
SET admitted_replicas = COALESCE(
        (SELECT jsonb_agg(COALESCE((role->>'replicas')::integer, 0))
         FROM jsonb_array_elements(spec->'roles') AS role),
        '[]'::jsonb),
    dispatched_replicas = COALESCE(
        (SELECT jsonb_agg(COALESCE((role->>'replicas')::integer, 0))
         FROM jsonb_array_elements(spec->'roles') AS role),
        '[]'::jsonb);

ALTER TABLE mlservices ALTER COLUMN phase SET DEFAULT 'Queued';

DROP INDEX IF EXISTS mlservices_sync_pending;
CREATE INDEX mlservices_sync_pending
    ON mlservices(id)
    WHERE admitted_replicas <> dispatched_replicas AND deleted_at IS NULL;

CREATE INDEX mlservices_admission_order
    ON mlservices(created_at ASC, id ASC)
    WHERE phase = 'Queued' AND deleted_at IS NULL;
