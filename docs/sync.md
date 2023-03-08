# sync

The sync command synchronizes a local directory tree to a Databricks workspace path.
The destination can be a repository (under `/Repos/<user>`) or a workspace path (under `/Users/<user>`).

By default it performs incremental synchronization where only changes since the last synchronization are applied.

Synchronization is **unidirectional**; changes to remote files are overwritten on a new invocation of the command.

Beware:
* Sync will not remove pre-existing remote files that do not exist in the local directory tree.
* Sync will overwrite pre-existing remote files if they exist in the local directory tree.

## Incremental synchronization

The sync command stores a synchronization snapshot file in the local directory tree under a `.databricks` directory.
This snapshot file contains state to compute which changes to the local directory tree have happened since the last synchronization.

To opt out of incremental synchronization and force a full synchronization, you can specify the `--full` argument.
This makes the command ignore any pre-existing snapshot and create a new one upon completion.

## Output

The sync command produces either text or JSON output.
Text output is intended to be human readable and prints the file names that the command operates on.
JSON output is intended to be machine readable.

### JSON output

If selected, this produces line-delimited JSON objects with a `type` field as discriminator.

Every time the command...
* checks the file system for changes, you'll see a `start` event.
* starts or completes a create/update/delete of a file, you'll see a `progress` event.
* completes a set of create/update/delete file operations, you'll see a `complete` event.

Every JSON object has a sequence number in the `seq` field that associates it with a synchronization run.

Progress events have a `progress` floating point number field between 0 and 1 indicating how far the operation has progressed.
A value of 0 means the operation has started and 1 means the operation has completed.
