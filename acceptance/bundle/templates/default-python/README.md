The 'serverless' and 'classic' directories contain full tests: they
have full output of materialized template, perform "bundle validate"
and in the future will perform deploy/summary/run.

Other directories (serverless-auto-\*) contain short tests: they only do
"bundle init" and then check that the output matches 'serverless' or 'classic' exactly.
