# libs/tmp

Temporary in-repo copy of the not-yet-published `github.com/databricks/sdk-go`
`files/v2` client and the two sibling modules it depends on. It exists only so
the CLI can build against `files/v2` before that module is published to the
Databricks Go proxy.

## Contents

| Path                | Upstream source (in the `universe` monorepo)     |
| ------------------- | ------------------------------------------------ |
| `files/`            | `deco/oss/repos/sdk-go/files`                    |
| `auth/`             | `deco/oss/repos/sdk-go/auth` (top-level package)  |
| `options/`          | `deco/oss/repos/sdk-go/options`                  |

Only the packages `files/v2` actually imports were copied: the top-level `auth`
package (not its `oidc`/`credentials`/`transport` subpackages) and
`options/{call,client,internaloptions,internal}`. The published
`github.com/databricks/sdk-go/core` module is a real dependency and is NOT
copied here; the vendored code imports it directly.

The only edits to the copied sources are the import-path rewrite from
`github.com/databricks/sdk-go/{files,auth,options}` to
`github.com/databricks/cli/libs/tmp/{files,auth,options}` (`core` imports are
left unchanged), plus two local fixes to the generated
`files/v2/client.go` that the upstream generator still gets wrong:

- Empty response bodies: the generated methods called `json.Unmarshal` on every
  response body, which fails with "unexpected end of JSON input" on the HEAD
  metadata calls and 204 PUT/DELETE responses that the Files API returns with no
  body. They now go through `unmarshalResponse` (in `files/v2/genhelper.go`),
  which treats an empty body as a zero-value response.
- HTTP method casing: the two directory/file metadata calls used the method
  string `"head"`; Go sends the method verbatim, so it is now `"HEAD"`.

Both are bugs in the upstream generator; report them there so the published
module needs no patching, and drop these edits when swapping back.

## Removing this copy

When `github.com/databricks/sdk-go/files/v2` is published:

1. Add it (and `auth`/`options` if they publish separately) to `go.mod`.
2. Rewrite every `github.com/databricks/cli/libs/tmp/{files,auth,options}`
   import back to `github.com/databricks/sdk-go/{files,auth,options}`. The
   `files/v2` package keeps its `v2` path element, so consumers only change the
   module prefix.
3. Delete `libs/tmp/`.
