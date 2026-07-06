CREATE SCHEMA IF NOT EXISTS integration;

CREATE TABLE integration.connection (
    id              uuid         NOT NULL  DEFAULT uuidv7()  PRIMARY KEY
    , workspace_id  uuid         NOT NULL  REFERENCES org.workspace (id)
    , provider      text         NOT NULL
    , settings      jsonb        NOT NULL  DEFAULT '{}'::jsonb
    , credentials   jsonb        NOT NULL  DEFAULT '{}'::jsonb
    , created_at    timestamptz  NOT NULL  DEFAULT now()
    , updated_at    timestamptz  NOT NULL  DEFAULT now()
    , deleted_at    timestamptz

    , CONSTRAINT cnst_integration_connection_valid_provider CHECK (provider = ANY (ARRAY['gitlab']))
);

CREATE UNIQUE INDEX idx_integration_connection_uniq_ws_provider ON integration.connection (workspace_id, provider) WHERE deleted_at IS NULL;

CREATE TABLE tracker.attachment (
    id                uuid         NOT NULL  DEFAULT uuidv7()  PRIMARY KEY
    , issue_id        uuid         NOT NULL  REFERENCES tracker.issue (id)
    , integration_id  uuid                   REFERENCES integration.connection (id)
    , source_type     text         NOT NULL
    , url             text         NOT NULL
    , title           text         NOT NULL
    , metadata        jsonb        NOT NULL  DEFAULT '{}'::jsonb
    , created_at      timestamptz  NOT NULL  DEFAULT now()
    , updated_at      timestamptz  NOT NULL  DEFAULT now()
    , deleted_at      timestamptz

    , CONSTRAINT cnst_tracker_attachment_valid_source CHECK (source_type = ANY (ARRAY['gitlab_mr', 'gitlab_commit']))
);

CREATE UNIQUE INDEX idx_tracker_attachment_uniq_url ON tracker.attachment (url, issue_id) WHERE deleted_at IS NULL;
CREATE INDEX        idx_tracker_attachment_issue_id ON tracker.attachment (issue_id);
