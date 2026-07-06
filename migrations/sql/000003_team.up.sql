CREATE TABLE org.invite (
    id              uuid         NOT NULL  DEFAULT uuidv7()  PRIMARY KEY
    , workspace_id  uuid         NOT NULL  REFERENCES org.workspace (id)
    , email         text         NOT NULL
    , role          text         NOT NULL
    , invited_by    uuid         NOT NULL  REFERENCES core.user (id)
    , token         text         NOT NULL
    , expires_at    timestamptz  NOT NULL
    , accepted_at   timestamptz
    , created_at    timestamptz  NOT NULL  DEFAULT now()
    , deleted_at    timestamptz

    , CONSTRAINT cnst_org_invite_valid_role CHECK (role = ANY (ARRAY['owner', 'admin', 'member']))
);

CREATE UNIQUE INDEX idx_org_invite_uniq_token        ON org.invite (token) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_org_invite_uniq_active_email ON org.invite (workspace_id, LOWER(email)) WHERE accepted_at IS NULL AND deleted_at IS NULL;

CREATE TABLE org.team (
    id                  uuid         NOT NULL  DEFAULT uuidv7()  PRIMARY KEY
    , workspace_id      uuid         NOT NULL  REFERENCES org.workspace (id)
    , name              text         NOT NULL
    , description       text
    , key               text         NOT NULL
    , next_issue_number int          NOT NULL  DEFAULT 1
    , created_at        timestamptz  NOT NULL  DEFAULT now()
    , updated_at        timestamptz  NOT NULL  DEFAULT now()
    , deleted_at        timestamptz
);

CREATE UNIQUE INDEX idx_org_team_uniq_key ON org.team (workspace_id, key) WHERE deleted_at IS NULL;

CREATE TABLE org.team_member (
    team_id      uuid         NOT NULL  REFERENCES org.team (id)
    , user_id    uuid         NOT NULL  REFERENCES core.user (id)
    , role       text         NOT NULL
    , created_at timestamptz  NOT NULL  DEFAULT now()
    , updated_at timestamptz  NOT NULL  DEFAULT now()
    , deleted_at timestamptz

    , PRIMARY KEY (team_id, user_id)
    , CONSTRAINT cnst_org_team_member_valid_role CHECK (role = ANY (ARRAY['owner', 'member']))
);
