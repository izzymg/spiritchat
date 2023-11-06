-- If the post has a parent, check the parent exists, and only in the same category.
CREATE OR REPLACE FUNCTION check_reply() RETURNS trigger AS $check_reply$
    BEGIN
        IF NOT NEW.parent = 0 THEN
            IF NOT EXISTS (SELECT FROM posts WHERE num = NEW.parent AND cat = NEW.cat) THEN
                RAISE EXCEPTION 'Nonexistent parent --> % on %', NEW.parent, NEW.cat USING ERRCODE = 23503;
            END IF;
        END IF;
        RETURN NEW;
    END;
$check_reply$ LANGUAGE plpgsql;


-- Create a new post, generating a category-specific number for it 
-- based on the most recent category number.
-- args: category, parent, content
-- Don't touch the ordering of this or it deadlocks under concurrent load.
CREATE OR REPLACE PROCEDURE write_post(TEXT, INTEGER, TEXT) AS $write_post$
    DECLARE
        post_num INTEGER;
    BEGIN
        SELECT post_count INTO post_num FROM cats WHERE tag = $1 FOR UPDATE;
        IF post_num IS NULL THEN
            RAISE EXCEPTION 'Nonexistent category --> %', $1 USING ERRCODE = 23503;
        END IF;
        INSERT INTO posts (cat, parent, content, num) VALUES (
            $1, $2, $3, post_num
        );
        UPDATE cats SET post_count = post_num + 1 WHERE tag = $1;
    END
$write_post$ LANGUAGE plpgsql;


-- Categories
CREATE TABLE IF NOT EXISTS cats (
    tag                     text,
    name                    text,
    post_count              integer NOT NULL DEFAULT 1,
    CONSTRAINT cat_tag      PRIMARY KEY(tag)
);

-- Posts
CREATE TABLE IF NOT EXISTS posts (
    num                     integer NOT NULL DEFAULT 0,
    cat                     text NOT NULL,
    parent                  integer NOT NULL,
    content                 text NOT NULL,
    created_at              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    --- Post must belong to a valid category and have a unique number for the category
    CONSTRAINT post_cat_num PRIMARY KEY(num, cat),
    FOREIGN KEY (cat)       REFERENCES cats (tag)         
);

-- Check replies before submission.
CREATE OR REPLACE TRIGGER check_reply BEFORE INSERT OR UPDATE ON posts
    FOR EACH ROW EXECUTE PROCEDURE check_reply();
