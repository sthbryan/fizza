CREATE UNIQUE INDEX uq_boards_default_per_project
    ON boards(project_id) WHERE is_default = 1;