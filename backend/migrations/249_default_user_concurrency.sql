-- Users with concurrency_is_default = true follow the global default_concurrency
-- setting. Other users keep their explicitly assigned concurrency, including 0.

DO $$
DECLARE
    v_default_concurrency INT;
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'users'
          AND column_name = 'concurrency_is_default'
    ) THEN
        ALTER TABLE users
            ADD COLUMN concurrency_is_default BOOLEAN NOT NULL DEFAULT FALSE;

        SELECT CASE
                   WHEN btrim(value) ~ '^[0-9]+$' AND btrim(value)::INT >= 1
                       THEN btrim(value)::INT
               END
        INTO v_default_concurrency
        FROM settings
        WHERE key = 'default_concurrency';

        v_default_concurrency := COALESCE(v_default_concurrency, 5);

        -- Historical intent was not stored. Classify by the current value once;
        -- later migrations must not reset an administrator's subsequent choices.
        UPDATE users
        SET concurrency_is_default = (concurrency = v_default_concurrency);
    END IF;
END $$;

-- Keep the cached API-key auth snapshot's effective concurrency in step with
-- single-user, batch, redeem and global-default updates.
CREATE OR REPLACE FUNCTION enqueue_user_auth_cache_invalidation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    target_user_id BIGINT;
BEGIN
    target_user_id := OLD.id;
    IF TG_OP = 'UPDATE'
       AND OLD.status IS NOT DISTINCT FROM NEW.status
       AND OLD.role IS NOT DISTINCT FROM NEW.role
       AND OLD.deleted_at IS NOT DISTINCT FROM NEW.deleted_at
       AND OLD.concurrency IS NOT DISTINCT FROM NEW.concurrency THEN
        RETURN NEW;
    END IF;

    INSERT INTO auth_cache_invalidation_outbox (cache_key)
    SELECT encode(sha256(convert_to(k.key, 'UTF8')), 'hex')
    FROM api_keys AS k
    WHERE k.user_id = target_user_id
      AND k.deleted_at IS NULL
      AND k.key <> '';
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;
