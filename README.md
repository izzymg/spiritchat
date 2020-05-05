# SpiritChat

SpiritChat is a chan-style imageboard/BBS written in Golang.

### Design

SpiritChat is broken up into a set of `categories` (cats). 

Users can post a `thread` in a category, and `reply` to those threads. Both of these are called `post` and share mostly the same data.

An algorithm is used to calculate a thread's `rate`. This will likely be based off of the number of replies the thread has, how recent the last replies to that thread were, and how old the thread is. 

This rate can be used to sort threads in the category.

Following flat BBS style, threads are inlined, meaning replies can only be placed
to the thread and not to other replies.

There will be a maximum amount of active threads per category. Once the maximum is
reached, each new created thread will cause deletion of the category's thread with the lowest `rate`.

There should also be a maximum number of replies per-thread, effectively locking that thread once the limit has been reached.

#### Brief note on rate

Rate could be evalulated several ways:

Rate will need to be known for each thread when a GET comes through for a cat's threads. As such it makes no sense to lazily calculate it on-demand.

Rate could be calculated via a sweep, where every `n` seconds, each thread is evaluated and has its rate updated.

It could also be calculated for each individual thread each time a variable in the rate calculation changes (number of replies), as well as each time a new thread is posted, as rate's should ideally steadily drop as new threads are created, assuming no new replies are being posted.

The ultimate goal of the rate system is to prioritise low-value, inactive threads, first and foremost when making room for new threads. New replies to a thread should "bump" the thread's rate up, inline with older forums.

### Brief note on anonymity

SpiritChat will be built from the ground up to provide **anonymity** to all users, requiring **no login or authentication** to begin posting. As such, anti-spam measures will need to be taken.

### Backend

SpiritChat will use **PostgreSQL** to store active post data.

Each post will need the minimum data set of:

* A uniquely identifying ID (UID)
* A "parent" UID, describing whether the post is a reply to a thread
* The contents of the post

It will be necessary to regularly calculate how many replies a thread has to it, which could be a very inefficient operation if done incorrectly.

The DB could contain a new table for each category, although dynamically creating tables does present potential safety issues and difficulty in keeping in sync.

Table size is not a large concern either, as there's a hard limit on the number of active threads and replies per category. To calculate the maximum amount of active posts (threads & replies) in the table would simply be:

`num cats * (max threads * max replies + max threads)`

e.g. 5 categories, 45 threads per category, 200 replies per thread = 45,225 max posts

So a single table makes the most sense for now. Each post could then have an optional field containing the number of replies to it, starting at zero. When a reply is made, it simply increments that field.

#### Post pipeline

The pipeline could function as follows:

Post thread:

```
HTTP/S POST [cat]

lookup [cat]

does not exist ? err 404

validate/sanitize failure ? err 400

write post () -> trigger potential thread deletion

```

Post reply:

```
HTTP/S POST [cat]/[uid]

lookup [cat]/[uid]

does not exist ? err 404

not a thread ? err 404

hit max replies ? err 400

validate/sanitize failure ? err 400

write post () -> trigger thread rate update

```

Write post:

```
generate UID

lock db

write post

unlock db

return 200
```

AFTER each new reply should trigger an update to the parent thread.

BEFORE each new thread should trigger a potential deletion of a lowest-rate thread.

These could be done in-code, or it could be done through DB hooks. Research needs to be done on the pros/cons.