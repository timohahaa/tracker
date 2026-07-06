CREATE SCHEMA IF NOT EXISTS org;

CREATE TABLE org.workspace (
    id            uuid         NOT NULL  DEFAULT uuidv7()  PRIMARY KEY
    , name        text         NOT NULL
    , slug        text         NOT NULL
    , created_at  timestamptz  NOT NULL  DEFAULT now()
    , updated_at  timestamptz  NOT NULL  DEFAULT now()
    , deleted_at  timestamptz
);

CREATE UNIQUE INDEX idx_org_workspace_uniq_slug ON org.workspace (LOWER(slug)) WHERE deleted_at IS NULL;

CREATE TABLE org.workspace_member (
    workspace_id  uuid         NOT NULL  REFERENCES org.workspace (id)
    , user_id     uuid         NOT NULL  REFERENCES core.user (id)
    , role        text         NOT NULL
    , username    text         NOT NULL
    , title       text
    , created_at  timestamptz  NOT NULL  DEFAULT now()
    , updated_at  timestamptz  NOT NULL  DEFAULT now()
    , deleted_at  timestamptz

    , PRIMARY KEY (workspace_id, user_id)
    , CONSTRAINT cnst_org_workspace_member_valid_role CHECK (role = ANY (ARRAY['owner', 'admin', 'member']))
);

CREATE UNIQUE INDEX idx_org_workspace_member_uniq_username ON org.workspace_member (workspace_id, LOWER(username)) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_org_workspace_member_uniq_owner    ON org.workspace_member (workspace_id) WHERE role = 'owner' AND deleted_at IS NULL;

CREATE TABLE core.api_token (
    id              uuid         NOT NULL  DEFAULT uuidv7()  PRIMARY KEY
    , user_id       uuid         NOT NULL  REFERENCES core.user (id)
    , workspace_id  uuid         NOT NULL  REFERENCES org.workspace (id)
    , name          text         NOT NULL
    , last_used_at  timestamptz
    , created_at    timestamptz  NOT NULL  DEFAULT now()
    , deleted_at    timestamptz
);

CREATE INDEX idx_core_api_token_user_id      ON core.api_token (user_id);
CREATE INDEX idx_core_api_token_workspace_id ON core.api_token (workspace_id);
