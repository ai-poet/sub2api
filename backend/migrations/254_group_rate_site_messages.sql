CREATE OR REPLACE FUNCTION group_rate_site_message_recipients(target_group_id BIGINT, at_time TIMESTAMPTZ)
RETURNS TABLE (user_id BIGINT, rate_multiplier NUMERIC)
LANGUAGE sql
STABLE
AS $$
    SELECT u.id, ugr.rate_multiplier
    FROM (
        SELECT k.user_id
        FROM api_keys k
        WHERE k.group_id = target_group_id
          AND k.deleted_at IS NULL
          AND k.status = 'active'
          AND (k.expires_at IS NULL OR k.expires_at > at_time)
        UNION
        SELECT s.user_id
        FROM user_subscriptions s
        WHERE s.group_id = target_group_id
          AND s.deleted_at IS NULL
          AND s.status = 'active'
          AND s.starts_at <= at_time
          AND s.expires_at > at_time
    ) recipients
    JOIN users u ON u.id = recipients.user_id
    JOIN groups g ON g.id = target_group_id
    LEFT JOIN user_group_rate_multipliers ugr ON ugr.user_id = u.id AND ugr.group_id = g.id
    WHERE u.deleted_at IS NULL
      AND u.status = 'active'
      AND g.deleted_at IS NULL
      AND g.status = 'active'
      AND (
          g.subscription_type = 'subscription'
          OR (NOT g.is_exclusive AND NOT u.restrict_public_groups)
          OR EXISTS (
              SELECT 1 FROM user_allowed_groups a
              WHERE a.user_id = u.id AND a.group_id = g.id
          )
      );
$$;

CREATE OR REPLACE FUNCTION group_rate_site_message_escape(value TEXT)
RETURNS TEXT
LANGUAGE plpgsql
IMMUTABLE
STRICT
AS $$
DECLARE
    normalized TEXT := left(regexp_replace(value, '[[:space:]]+', ' ', 'g'), 120);
    escaped TEXT := '';
    letter TEXT;
    position INTEGER;
BEGIN
    FOR position IN 1..char_length(normalized) LOOP
        letter := substr(normalized, position, 1);
        IF strpos(chr(92) || '`*_{}[]()#+-.!|<>~', letter) > 0 THEN
            escaped := escaped || chr(92);
        END IF;
        escaped := escaped || letter;
    END LOOP;
    RETURN escaped;
END;
$$;

CREATE OR REPLACE FUNCTION insert_group_rate_site_message(
    recipient_id BIGINT,
    previous_group groups,
    current_group groups,
    previous_override NUMERIC,
    current_override NUMERIC,
    changed_at TIMESTAMPTZ,
    personal_change BOOLEAN
)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    previous_base NUMERIC := COALESCE(previous_override, previous_group.rate_multiplier);
    current_base NUMERIC := COALESCE(current_override, current_group.rate_multiplier);
    previous_image NUMERIC;
    current_image NUMERIC;
    previous_video NUMERIC;
    current_video NUMERIC;
    zh_rows TEXT := '';
    en_rows TEXT := '';
    group_name TEXT := group_rate_site_message_escape(current_group.name);
    saved_at TEXT := to_char(changed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') || ' UTC';
    message_title TEXT;
    message_source TEXT;
BEGIN
    previous_image := CASE WHEN previous_group.image_rate_independent
        THEN GREATEST(0, previous_group.image_rate_multiplier) ELSE previous_base END;
    current_image := CASE WHEN current_group.image_rate_independent
        THEN GREATEST(0, current_group.image_rate_multiplier) ELSE current_base END;
    previous_video := CASE WHEN previous_group.video_rate_independent
        THEN GREATEST(0, previous_group.video_rate_multiplier) ELSE previous_base END;
    current_video := CASE WHEN current_group.video_rate_independent
        THEN GREATEST(0, current_group.video_rate_multiplier) ELSE current_base END;

    IF previous_base IS DISTINCT FROM current_base THEN
        zh_rows := zh_rows || format(E'| 基础倍率（不含高峰加成） | %s× | %s× |\n', trim_scale(previous_base), trim_scale(current_base));
        en_rows := en_rows || format(E'| Base multiplier (before peak adjustment) | %s× | %s× |\n', trim_scale(previous_base), trim_scale(current_base));
    END IF;
    IF previous_image IS DISTINCT FROM current_image
       OR previous_group.image_rate_independent IS DISTINCT FROM current_group.image_rate_independent THEN
        zh_rows := zh_rows || format(E'| 图片倍率 | %s×（%s） | %s×（%s） |\n',
            trim_scale(previous_image), CASE WHEN previous_group.image_rate_independent THEN '独立' ELSE '跟随基础' END,
            trim_scale(current_image), CASE WHEN current_group.image_rate_independent THEN '独立' ELSE '跟随基础' END);
        en_rows := en_rows || format(E'| Image multiplier | %s× (%s) | %s× (%s) |\n',
            trim_scale(previous_image), CASE WHEN previous_group.image_rate_independent THEN 'independent' ELSE 'follows base' END,
            trim_scale(current_image), CASE WHEN current_group.image_rate_independent THEN 'independent' ELSE 'follows base' END);
    END IF;
    IF previous_video IS DISTINCT FROM current_video
       OR previous_group.video_rate_independent IS DISTINCT FROM current_group.video_rate_independent THEN
        zh_rows := zh_rows || format(E'| 视频倍率 | %s×（%s） | %s×（%s） |\n',
            trim_scale(previous_video), CASE WHEN previous_group.video_rate_independent THEN '独立' ELSE '跟随基础' END,
            trim_scale(current_video), CASE WHEN current_group.video_rate_independent THEN '独立' ELSE '跟随基础' END);
        en_rows := en_rows || format(E'| Video multiplier | %s× (%s) | %s× (%s) |\n',
            trim_scale(previous_video), CASE WHEN previous_group.video_rate_independent THEN 'independent' ELSE 'follows base' END,
            trim_scale(current_video), CASE WHEN current_group.video_rate_independent THEN 'independent' ELSE 'follows base' END);
    END IF;
    IF zh_rows = '' THEN
        RETURN;
    END IF;

    IF personal_change THEN
        message_title := '专属倍率调整通知 / Personal rate update';
        message_source := 'user_group_rate_changed';
    ELSE
        message_title := '分组倍率调整通知 / Group rate update';
        message_source := 'group_rate_changed';
    END IF;

    INSERT INTO site_messages (user_id, category, title, content, source_type, source_id, created_at)
    VALUES (
        recipient_id,
        'system',
        message_title,
        format(E'您使用的分组「%s」计费倍率已调整。\n\n调整时间：%s\n\n| 项目 | 调整前 | 调整后 |\n| --- | --- | --- |\n%s\n配置已保存，新倍率将在相关计费缓存刷新后生效；高峰加成等其他计费规则仍按分组配置执行。当前价格请查看[模型广场](/models)。\n\n---\n\nBilling multipliers for your group “%s” have changed.\n\nUpdated at: %s\n\n| Item | Before | After |\n| --- | --- | --- |\n%s\nThe configuration has been saved. New multipliers take effect when the relevant billing caches refresh; other billing rules, including peak adjustments, still apply according to the group configuration. See the [model catalog](/models) for current prices.',
            group_name, saved_at, zh_rows, group_name, saved_at, en_rows),
        message_source,
        format('group:%s:tx:%s', current_group.id, txid_current()),
        changed_at
    );
END;
$$;

CREATE OR REPLACE FUNCTION notify_group_rate_site_messages()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    recipient RECORD;
    changed_at TIMESTAMPTZ := statement_timestamp();
BEGIN
    IF NEW.deleted_at IS NOT NULL OR NEW.status <> 'active' THEN
        RETURN NULL;
    END IF;
    FOR recipient IN
        SELECT * FROM group_rate_site_message_recipients(NEW.id, changed_at)
    LOOP
        PERFORM insert_group_rate_site_message(
            recipient.user_id, OLD, NEW, recipient.rate_multiplier, recipient.rate_multiplier, changed_at, false);
    END LOOP;
    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trg_groups_rate_site_messages ON groups;
CREATE TRIGGER trg_groups_rate_site_messages
AFTER UPDATE OF rate_multiplier, image_rate_independent, image_rate_multiplier, video_rate_independent, video_rate_multiplier
ON groups
FOR EACH ROW
WHEN (
    OLD.rate_multiplier IS DISTINCT FROM NEW.rate_multiplier
    OR OLD.image_rate_independent IS DISTINCT FROM NEW.image_rate_independent
    OR OLD.image_rate_multiplier IS DISTINCT FROM NEW.image_rate_multiplier
    OR OLD.video_rate_independent IS DISTINCT FROM NEW.video_rate_independent
    OR OLD.video_rate_multiplier IS DISTINCT FROM NEW.video_rate_multiplier
)
EXECUTE FUNCTION notify_group_rate_site_messages();

CREATE OR REPLACE FUNCTION notify_user_group_rate_site_message()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    target_user_id BIGINT;
    target_group_id BIGINT;
    previous_override NUMERIC;
    current_override NUMERIC;
    target_group groups%ROWTYPE;
    changed_at TIMESTAMPTZ := statement_timestamp();
BEGIN
    IF TG_OP = 'DELETE' THEN
        target_user_id := OLD.user_id;
        target_group_id := OLD.group_id;
        previous_override := OLD.rate_multiplier;
    ELSIF TG_OP = 'INSERT' THEN
        target_user_id := NEW.user_id;
        target_group_id := NEW.group_id;
        current_override := NEW.rate_multiplier;
    ELSE
        target_user_id := NEW.user_id;
        target_group_id := NEW.group_id;
        previous_override := OLD.rate_multiplier;
        current_override := NEW.rate_multiplier;
    END IF;
    IF previous_override IS NOT DISTINCT FROM current_override THEN
        RETURN NULL;
    END IF;

    SELECT * INTO target_group FROM groups
    WHERE id = target_group_id AND deleted_at IS NULL AND status = 'active';
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM group_rate_site_message_recipients(target_group_id, changed_at) r
        WHERE r.user_id = target_user_id
    ) THEN
        RETURN NULL;
    END IF;
    PERFORM insert_group_rate_site_message(
        target_user_id, target_group, target_group, previous_override, current_override, changed_at, true);
    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trg_user_group_rates_site_messages ON user_group_rate_multipliers;
CREATE TRIGGER trg_user_group_rates_site_messages
AFTER INSERT OR UPDATE OR DELETE ON user_group_rate_multipliers
FOR EACH ROW EXECUTE FUNCTION notify_user_group_rate_site_message();
