-- Categories
CREATE TABLE IF NOT EXISTS cats (
    tag                     text,
    name                    text NOT NULL DEFAULT '',
    description             text NOT NULL DEFAULT '',
    post_count              integer NOT NULL DEFAULT 1,
    CONSTRAINT cat_tag      PRIMARY KEY(tag)
);

-- Posts
CREATE TABLE IF NOT EXISTS posts (
    num                     integer NOT NULL DEFAULT 0,
    cat                     text NOT NULL,
    subject                 text NOT NULL,
    parent                  integer NOT NULL,
    content                 text NOT NULL,
    username                text NOT NULL,
    email                   text NOT NULL,
    ip                      text NOT NULL,
    created_at              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    --- Post must belong to a valid category and have a unique number for the category
    CONSTRAINT post_cat_num PRIMARY KEY(num, cat),
    FOREIGN KEY (cat)       REFERENCES cats (tag)         
);

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

-- Check replies before submission.
CREATE OR REPLACE TRIGGER check_reply BEFORE INSERT OR UPDATE ON posts
    FOR EACH ROW EXECUTE PROCEDURE check_reply();


-- Create a new post, generating a category-specific number for it 
-- based on the most recent category number.
-- args: category, parent, content, subject, username, email, ip
-- Don't touch the ordering of this or it deadlocks under concurrent load.
CREATE OR REPLACE PROCEDURE write_post(TEXT, INTEGER, TEXT, TEXT, TEXT, TEXT, TEXT) AS $write_post$
    DECLARE
        post_num INTEGER;
    BEGIN
        SELECT post_count INTO post_num FROM cats WHERE tag = $1 FOR UPDATE;
        IF post_num IS NULL THEN
            RAISE EXCEPTION 'Nonexistent category --> %', $1 USING ERRCODE = 23503;
        END IF;
        INSERT INTO posts (cat, parent, content, num, subject, username, email, ip) VALUES (
            $1, $2, $3, post_num, $4, $5, $6, $7
        );
        UPDATE cats SET post_count = post_num + 1 WHERE tag = $1;
    END
$write_post$ LANGUAGE plpgsql;

-- Drop posts that don't have a parent anymore
CREATE OR REPLACE FUNCTION drop_orphans() RETURNS trigger as $drop_orphans$
    BEGIN
        IF OLD.parent = 0 THEN
            DELETE FROM posts WHERE cat = OLD.cat AND parent = old.num;
            RETURN OLD;
        END IF;
        RETURN NULL;
    END
$drop_orphans$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER drop_orphans
    AFTER DELETE ON posts
    FOR EACH ROW EXECUTE FUNCTION drop_orphans();

-- Drop all posts on a category
CREATE OR REPLACE FUNCTION drop_category_posts() RETURNS TRIGGER as $drop_category_posts$
    BEGIN
        DELETE FROM posts WHERE cat = OLD.tag;
        RETURN OLD;
    END
$drop_category_posts$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER drop_category_posts
    BEFORE DELETE ON cats
    FOR EACH ROW EXECUTE FUNCTION drop_category_posts();