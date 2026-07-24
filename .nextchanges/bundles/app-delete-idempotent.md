Fixed `bundle deploy`/`bundle destroy` failing when an app is still in the transient DELETING state; the delete is now treated as complete instead of erroring (direct engine only).
