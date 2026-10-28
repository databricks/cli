* Fixed `bundle validate` failing with "expected a map to index" when a job `schedule`, `trigger` or `continuous` is a reference to a whole block, such as `${resources.jobs.base.schedule}`.
