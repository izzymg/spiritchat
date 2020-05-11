ALTER TABLE cats ADD COLUMN post_count integer NOT NULL DEFAULT 1;

-- If the post has a parent, check the parent exists, and only in the same category.
CREATE OR REPLACE FUNCTION check_reply() RETURNS trigger AS $check_reply$
    BEGIN
        IF NOT NEW.parent IS NULL THEN
            IF NOT EXISTS (SELECT FROM posts WHERE uid = NEW.parent AND cat = NEW.cat) THEN
                RAISE EXCEPTION 'Reply does not have a corresponding parent';
            END IF;
        END IF;
        RETURN NEW;
    END;
$check_reply$ LANGUAGE plpgsql;


-- Create a new post, generating a category-specific number for it 
-- based on the most recent category number.
-- args: uid, category, parent, content
CREATE OR REPLACE PROCEDURE write_post(TEXT, TEXT, TEXT, TEXT) AS $write_post$
    BEGIN
        INSERT INTO posts (uid, cat, parent, content, num) VALUES (
            $1, $2, NULLIF($3, ''), $4, COALESCE(
                (SELECT post_count FROM cats WHERE name = $2 FOR UPDATE),
                1
            )
        );
        UPDATE cats SET post_count = post_count + 1;
    END
$write_post$ LANGUAGE plpgsql;
