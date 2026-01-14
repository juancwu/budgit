-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS space_invitations (
    token TEXT PRIMARY KEY NOT NULL,
    space_id TEXT NOT NULL,
    inviter_id TEXT NOT NULL,
    email TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'accepted', 'expired')),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE,
    FOREIGN KEY (inviter_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_space_invitations_email ON space_invitations(email);
CREATE INDEX IF NOT EXISTS idx_space_invitations_space_id ON space_invitations(space_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_space_invitations_space_id;
DROP INDEX IF EXISTS idx_space_invitations_email;
DROP TABLE IF EXISTS space_invitations;
-- +goose StatementEnd
