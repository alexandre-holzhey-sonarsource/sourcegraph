BEGIN;
ALTER TABLE threads DROP COLUMN "author_user_id";
ALTER TABLE threads DROP COLUMN "html_lines_before";
ALTER TABLE threads DROP COLUMN "html_lines";
ALTER TABLE threads DROP COLUMN "html_lines_after";
ALTER TABLE threads DROP COLUMN "text_lines_before";
ALTER TABLE threads DROP COLUMN "text_lines";
ALTER TABLE threads DROP COLUMN "text_lines_after";
ALTER TABLE threads DROP COLUMN "text_lines_selection_range_start";
ALTER TABLE threads DROP COLUMN "text_lines_selection_range_length";
COMMIT;
