Emptying a resource list now keeps enforcing it. A bundle declaring `grants: []` revokes
any grant added out of band on every deploy, not just the first.
