-- Durable MLRun admission queue. The priority snapshot is parsed from the
-- public annotation at create time; scheduled_at records the first successful
-- runtime submission and remains stable for the lifetime of the run.
ALTER TABLE mlruns
    ADD COLUMN priority integer NOT NULL DEFAULT 0,
    ADD COLUMN scheduled_at timestamptz;

ALTER TABLE mlruns ALTER COLUMN phase SET DEFAULT 'Queued';

-- Preserve priority for pre-migration rows that already carried a valid
-- annotation. Invalid legacy values remain neutral instead of blocking the
-- migration; all new writes are rejected at the API boundary.
UPDATE mlruns
SET priority = (annotations->>'scheduling.axisml.io/priority')::integer
WHERE annotations ? 'scheduling.axisml.io/priority'
  AND annotations->>'scheduling.axisml.io/priority' ~ '^[+-]?[0-9]+$'
  AND (annotations->>'scheduling.axisml.io/priority')::numeric
      BETWEEN -2147483648 AND 2147483647;

CREATE INDEX mlruns_queue_order
    ON mlruns(priority DESC, created_at ASC, id ASC)
    WHERE phase = 'Queued' AND deleted_at IS NULL;
