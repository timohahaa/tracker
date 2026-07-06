CREATE SCHEMA IF NOT EXISTS core;

CREATE TABLE core.user (
    id               uuid         NOT NULL  DEFAULT uuidv7()  PRIMARY KEY
    , email          text         NOT NULL
    , password_hash  text         NOT NULL
    , display_name   text         NOT NULL
    , created_at     timestamptz  NOT NULL  DEFAULT now()
    , updated_at     timestamptz  NOT NULL  DEFAULT now()
    , deleted_at     timestamptz
);

CREATE UNIQUE INDEX idx_core_user_uniq_email ON core.user (LOWER(email)) WHERE deleted_at IS NULL;

CREATE TABLE core.session (
    id            text         NOT NULL  PRIMARY KEY
    , user_id     uuid         NOT NULL  REFERENCES core.user (id)
    , expires_at  timestamptz  NOT NULL
    , created_at  timestamptz  NOT NULL  DEFAULT now()
);

CREATE INDEX idx_core_session_user_id ON core.session (user_id);
