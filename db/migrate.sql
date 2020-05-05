CREATE TABLE cats (
    name                    text,
    CONSTRAINT cat_name     PRIMARY KEY(name)
);

CREATE TABLE posts (
    uid                     text,
    cat                     text,
    parent                  text,
    content                 text NOT NULL,
    CONSTRAINT post_uid     PRIMARY KEY (uid),
    FOREIGN KEY (cat)       REFERENCES cats (name)                
);

-- If the post has a parent, check the parent exists.
CREATE FUNCTION check_reply() RETURNS trigger AS $check_reply$
    BEGIN
        IF NOT NEW.parent IS NULL THEN
            IF NOT EXISTS (SELECT FROM posts WHERE uid = NEW.parent) THEN
                RAISE EXCEPTION 'Reply does not have a corresponding parent';
            END IF;
        END IF;
        RETURN NEW;
    END;
$check_reply$ LANGUAGE plpgsql;

-- Check reply should happen before submission.
CREATE TRIGGER check_reply BEFORE INSERT OR UPDATE ON posts
    FOR EACH ROW EXECUTE PROCEDURE check_reply();