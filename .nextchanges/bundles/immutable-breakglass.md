* Immutable-folder deploys now detect a break-glassed snapshot and refuse to redeploy over it; `bundle deploy --force` recovers by creating a new snapshot at a suffixed path.
