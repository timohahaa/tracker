CREATE SCHEMA IF NOT EXISTS tracker;

CREATE TABLE tracker.project (
    id              uuid         NOT NULL  DEFAULT uuidv7()  PRIMARY KEY
    , workspace_id  uuid         NOT NULL  REFERENCES org.workspace (id)
    , lead_id       uuid                   REFERENCES core.user (id)
    , name          text         NOT NULL
    , description   text
    , status        text         NOT NULL  DEFAULT 'planned'
    , created_at    timestamptz  NOT NULL  DEFAULT now()
    , updated_at    timestamptz  NOT NULL  DEFAULT now()
    , deleted_at    timestamptz

    , CONSTRAINT cnst_tracker_project_valid_status CHECK (status = ANY (ARRAY['planned', 'in_progress', 'completed', 'canceled']))
);

CREATE INDEX idx_tracker_project_workspace_id ON tracker.project (workspace_id);

CREATE TABLE tracker.project_team (
    project_id   uuid         NOT NULL  REFERENCES tracker.project (id)
    , team_id    uuid         NOT NULL  REFERENCES org.team (id)
    , created_at timestamptz  NOT NULL  DEFAULT now()
    , deleted_at timestamptz

    , PRIMARY KEY (project_id, team_id)
);

CREATE INDEX idx_tracker_project_team_team_id ON tracker.project_team (team_id);

CREATE TABLE tracker.status (
    id             uuid         NOT NULL  DEFAULT uuidv7()  PRIMARY KEY
    , team_id      uuid         NOT NULL  REFERENCES org.team (id)
    , name         text         NOT NULL
    , status_group text         NOT NULL
    , position     int          NOT NULL
    , created_at   timestamptz  NOT NULL  DEFAULT now()
    , updated_at   timestamptz  NOT NULL  DEFAULT now()
    , deleted_at   timestamptz

    , CONSTRAINT cnst_tracker_status_valid_group CHECK (status_group = ANY (ARRAY['backlog', 'planned', 'in_progress', 'completed', 'canceled']))
);

CREATE UNIQUE INDEX idx_tracker_status_uniq_position ON tracker.status (team_id, status_group, position) WHERE deleted_at IS NULL;

CREATE TABLE tracker.issue (
    id             uuid         NOT NULL  DEFAULT uuidv7()  PRIMARY KEY
    , team_id      uuid         NOT NULL  REFERENCES org.team (id)
    , project_id   uuid                   REFERENCES tracker.project (id)
    , status_id    uuid         NOT NULL  REFERENCES tracker.status (id)
    , creator_id   uuid                   REFERENCES core.user (id)
    , assignee_id  uuid                   REFERENCES core.user (id)
    , title        text         NOT NULL
    , description  text
    , priority     text         NOT NULL  DEFAULT 'none'
    , due_date     date
    , issue_number int          NOT NULL
    , created_at   timestamptz  NOT NULL  DEFAULT now()
    , updated_at   timestamptz  NOT NULL  DEFAULT now()
    , deleted_at   timestamptz

    , CONSTRAINT cnst_tracker_issue_valid_priority CHECK (priority = ANY (ARRAY['none', 'urgent', 'high', 'medium', 'low']))
);

CREATE UNIQUE INDEX idx_tracker_issue_uniq_number ON tracker.issue (team_id, issue_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_tracker_issue_status_id   ON tracker.issue (status_id);
CREATE INDEX idx_tracker_issue_assignee_id ON tracker.issue (assignee_id);
CREATE INDEX idx_tracker_issue_project_id  ON tracker.issue (project_id);

CREATE TABLE tracker.label (
    id              uuid         NOT NULL  DEFAULT uuidv7()  PRIMARY KEY
    , workspace_id  uuid         NOT NULL  REFERENCES org.workspace (id)
    , team_id       uuid                   REFERENCES org.team (id)
    , name          text         NOT NULL
    , color         text         NOT NULL
    , created_at    timestamptz  NOT NULL  DEFAULT now()
    , updated_at    timestamptz  NOT NULL  DEFAULT now()
    , deleted_at    timestamptz
);

CREATE INDEX idx_tracker_label_workspace_id ON tracker.label (workspace_id);

CREATE UNIQUE INDEX idx_tracker_label_uniq_ws_name   ON tracker.label (workspace_id, name) WHERE team_id IS NULL     AND deleted_at IS NULL;
CREATE UNIQUE INDEX idx_tracker_label_uniq_team_name ON tracker.label (team_id, name)      WHERE team_id IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE tracker.issue_label (
    issue_id     uuid         NOT NULL  REFERENCES tracker.issue (id)
    , label_id   uuid         NOT NULL  REFERENCES tracker.label (id)
    , created_at timestamptz  NOT NULL  DEFAULT now()
    , deleted_at timestamptz

    , PRIMARY KEY (issue_id, label_id)
);

CREATE INDEX idx_tracker_issue_label_label_id ON tracker.issue_label (label_id);
