Reject empty, blank, or control-character resource identifiers and incomplete pipeline library paths during bundle validation.

Configs that previously only warned on missing names (for example models and apps) now fail validate. Explicit empty strings for UC parent fields such as catalog_name are rejected; omitted parents still warn.
