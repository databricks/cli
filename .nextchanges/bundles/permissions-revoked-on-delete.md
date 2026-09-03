Emptying a `permissions` list, or removing the block, now revokes every permission except the
object owner, which the API requires. Previously both were silently ignored.
