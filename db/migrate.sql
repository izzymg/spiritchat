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
-- based on the most recent category number + 1, or 1.
-- Parent is set to null if given an empty string ('').
-- args: uid, category, parent, content
CREATE OR REPLACE PROCEDURE write_post(TEXT, TEXT, TEXT, TEXT) AS $write_post$
    BEGIN
        INSERT INTO posts (uid, cat, parent, content, num) VALUES (
            $1, $2, NULLIF($3, ''), $4, COALESCE(
                (SELECT num FROM posts WHERE cat = $2 ORDER BY created_at DESC LIMIT 1) + 1,
                1
            )
        );
    END
$write_post$ LANGUAGE plpgsql;


-- Categories
CREATE TABLE cats (
    name                    text,
    CONSTRAINT cat_name     PRIMARY KEY(name)
);

-- Posts
CREATE TABLE posts (
    uid                     text,
    num                     integer NOT NULL DEFAULT 0,
    cat                     text NOT NULL,
    parent                  text,
    content                 text NOT NULL,
    created_at              timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    --- Post number should be unique to the category
    UNIQUE                  (cat, num),
    --- Post UID is the primary key
    CONSTRAINT post_uid     PRIMARY KEY (uid),
    --- Post must belong to a valid category
    FOREIGN KEY (cat)       REFERENCES cats (name)         
);

-- Check reply should happen before submission.
CREATE OR REPLACE TRIGGER check_reply BEFORE INSERT OR UPDATE ON posts
    FOR EACH ROW EXECUTE PROCEDURE check_reply();