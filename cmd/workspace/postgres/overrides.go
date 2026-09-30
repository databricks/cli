package postgres

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/duration"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/spf13/cobra"
)

// createRoleOverride appends an example body to the auto-generated help and
// rejects wrapped {"role": ...} bodies with a clear client-side error.
// The --json flag binds to the inner Role object (CreateRoleRequest.Role,
// JSON-tagged "role"), so users supply spec/name/etc. directly. Without an
// example, the auto-generated `// TODO: complex arg: spec` flags leave no
// hint about the body shape and the API's "Field 'role' is required" error
// is unhelpful when the request body is wrong.
func createRoleOverride(createRoleCmd *cobra.Command, _ *postgres.CreateRoleRequest) {
	prevPreRunE := createRoleCmd.PreRunE
	createRoleCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if err := rejectWrappedRoleJSON(cmd); err != nil {
			return err
		}
		if prevPreRunE != nil {
			return prevPreRunE(cmd, args)
		}
		return nil
	}

	createRoleCmd.Long += `

  Body shape (passed via --json): fields go directly on the Role object.
  Do not wrap them in '{"role": ...}' — the CLI rejects wrapped bodies
  client-side with a hint pointing to the right shape.

  Example — create a service-principal-backed role:

    databricks postgres create-role projects/<PROJECT_ID>/branches/<BRANCH_ID> \
      --role-id <SP_CLIENT_ID> \
      --json '{"spec": {"identity_type": "SERVICE_PRINCIPAL", "postgres_role": "<SP_CLIENT_ID>", "auth_method": "LAKEBASE_OAUTH_V1"}}'

  The example omits 'membership_roles' so the role starts with default
  privileges only — grant database/schema/table access separately via
  SQL, following least privilege. Set 'membership_roles' (e.g.
  ["DATABRICKS_SUPERUSER"]) only when broad administrative access is
  intentional.

  See databricks-sdk-go/service/postgres.RoleRoleSpec for the full set of
  spec fields.`
}

// rejectWrappedRoleJSON returns a clear error when --json is a top-level
// object containing a "role" key. Without this guard the generated unmarshal
// strips the unknown outer "role" field with a warning and ships an empty
// body, and the server rejects with a confusing "Field 'role' is required"
// message.
func rejectWrappedRoleJSON(cmd *cobra.Command) error {
	jf, err := postgresJSONFlag(cmd, "create-role")
	if err != nil {
		return err
	}
	return jf.RejectWrappedJSON("role", `databricks postgres create-role projects/<PROJECT_ID>/branches/<BRANCH_ID> \
    --role-id <SP_CLIENT_ID> \
    --json '{"spec": {"identity_type": "SERVICE_PRINCIPAL", "postgres_role": "<SP_CLIENT_ID>", "auth_method": "LAKEBASE_OAUTH_V1"}}'`)
}

// createBranchOverride adds --ttl and --no-expiry flags to create-branch, so
// setting a branch's expiration doesn't need a hand-written --json spec (the
// generated command leaves the nested spec to --json). It sets req.Branch.Spec in
// PreRunE, before the generated RunE merges --json and calls the API — the same
// way the generated --name flag pre-populates the request.
func createBranchOverride(cmd *cobra.Command, req *postgres.CreateBranchRequest) {
	var ttl string
	var noExpiry bool
	cmd.Flags().StringVar(&ttl, "ttl", "", "Relative time-to-live before the branch expires, e.g. 604800s, 168h, or 7d (7 days). Required unless --no-expiry.")
	cmd.Flags().BoolVar(&noExpiry, "no-expiry", false, "Create the branch with no expiration. Required unless --ttl.")

	prevPreRunE := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		var jsonBranch postgres.Branch
		if cmd.Flags().Changed("json") {
			jf, err := postgresJSONFlag(cmd, "create-branch")
			if err != nil {
				return err
			}
			// Decode --json into the SDK type via the same path the generated
			// RunE uses, so expiration detection can't drift from the real merge.
			if diags := jf.Unmarshal(&jsonBranch); diags.HasError() {
				return diags.Error()
			}
		}

		spec, err := reconcileBranchExpiration(req.Branch.Spec, cmd.Flags().Changed("ttl"), ttl, cmd.Flags().Changed("no-expiry"), noExpiry, specHasExpiration(jsonBranch.Spec))
		if err != nil {
			return err
		}
		req.Branch.Spec = spec

		if prevPreRunE != nil {
			return prevPreRunE(cmd, args)
		}
		return nil
	}

	cmd.Long += `

  Branch expiration (required — set exactly one):
    --ttl <duration>   relative time-to-live; sets spec.ttl. Accepts the REST API
                       form (604800s), a Go duration (168h), or day/week units
                       (7d, 3w) — 604800s, 168h and 7d all mean 7 days.
    --no-expiry        the branch never expires (sets spec.no_expiry).

  You must set --ttl, --no-expiry, or an expiration inside --json (spec.ttl /
  spec.expire_time / spec.no_expiry). These are mutually exclusive.

  Examples:

    # Expire after 7 days
    databricks postgres create-branch projects/<PROJECT_ID> <BRANCH_ID> --ttl 604800s

    # Never expire
    databricks postgres create-branch projects/<PROJECT_ID> <BRANCH_ID> --no-expiry`
}

// postgresJSONFlag returns the --json flag for a generated postgres create
// command, or a loud internal error if it is missing or mistyped (a codegen
// change would be the cause). cmdName names the command for the error message.
func postgresJSONFlag(cmd *cobra.Command, cmdName string) (*flags.JsonFlag, error) {
	flag := cmd.Flags().Lookup("json")
	if flag == nil {
		return nil, fmt.Errorf("internal: postgres %s expected a --json flag; this override is wired to the wrong command", cmdName)
	}
	jf, ok := flag.Value.(*flags.JsonFlag)
	if !ok {
		return nil, fmt.Errorf("internal: postgres %s --json flag has unexpected type %T; expected *flags.JsonFlag", cmdName, flag.Value)
	}
	return jf, nil
}

// reconcileBranchExpiration builds the BranchSpec from the --ttl / --no-expiry
// flags before --json is merged; jsonHasExpiration reports whether --json already
// sets one. Exactly one expiration source is required, and the three are mutually
// exclusive (the API rejects no_expiry=false and multiple sources). When the
// expiration comes from --json alone, spec is left for the generated RunE to merge.
func reconcileBranchExpiration(spec *postgres.BranchSpec, ttlChanged bool, ttl string, noExpiryChanged, noExpiry, jsonHasExpiration bool) (*postgres.BranchSpec, error) {
	// --no-expiry=false asks for "expiry on" without saying how, which the API
	// rejects; point the user at the flag that expresses intent.
	if noExpiryChanged && !noExpiry {
		return nil, errors.New("--no-expiry=false is not valid; set --ttl <duration> such as 604800s, or spec.expire_time via --json")
	}

	flagTTL := ttlChanged
	flagNoExpiry := noExpiryChanged && noExpiry

	sources := 0
	for _, set := range []bool{flagTTL, flagNoExpiry, jsonHasExpiration} {
		if set {
			sources++
		}
	}
	switch {
	case sources == 0:
		return nil, errors.New("a branch expiration is required; set --ttl <duration> such as 604800s, --no-expiry, or a spec expiration in --json")
	case sources > 1:
		return nil, errors.New("branch expiration set more than once; use exactly one of --ttl, --no-expiry, or a spec expiration in --json")
	}

	switch {
	case flagTTL:
		d, err := parseTTL(ttl)
		if err != nil {
			return nil, err
		}
		if spec == nil {
			spec = &postgres.BranchSpec{}
		}
		spec.Ttl = d
	case flagNoExpiry:
		if spec == nil {
			spec = &postgres.BranchSpec{}
		}
		spec.NoExpiry = true
	case jsonHasExpiration:
		// Expiration comes from --json; leave spec for the generated RunE to merge.
	}
	return spec, nil
}

// specHasExpiration reports whether a decoded BranchSpec sets an expiration:
// spec.ttl, spec.expire_time, or spec.no_expiry. no_expiry present as false still
// counts as "set" (detected via ForceSendFields, which the SDK decode populates
// for explicit zero values), so combining it with --no-expiry is caught as a
// conflict instead of the json silently overriding the flag on the wire.
func specHasExpiration(s *postgres.BranchSpec) bool {
	return s != nil && (s.Ttl != nil || s.ExpireTime != nil || s.NoExpiry ||
		slices.Contains(s.ForceSendFields, "NoExpiry"))
}

// parseTTL parses the --ttl value into a *duration.Duration. It accepts the REST
// API's protobuf-duration form ("604800s"), Go-style durations ("168h"), and the
// day/week extension ("7d", "3w"); duration.New re-serializes to the API form.
func parseTTL(s string) (*duration.Duration, error) {
	if s == "" {
		return nil, errors.New("--ttl must not be empty; use e.g. 604800s, 168h, or 7d for 7 days")
	}
	d, err := parseTTLDuration(s)
	if err != nil {
		return nil, fmt.Errorf("invalid --ttl %q: %w; use e.g. 604800s, 168h, 7d, or 3w", s, err)
	}
	if d <= 0 {
		return nil, fmt.Errorf("invalid --ttl %q: must be a positive duration", s)
	}
	return duration.New(d), nil
}

// ttlComponent matches one leading "<number><unit>" component of a duration
// string, including the day (d) and week (w) units time.ParseDuration lacks.
var ttlComponent = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)(ns|us|µs|ms|s|m|h|d|w)`)

const (
	hoursPerDay  = 24
	hoursPerWeek = 24 * 7
	// maxDurationHours is the largest whole-hour count representable as a
	// time.Duration (an int64 nanosecond count).
	maxDurationHours = int64(math.MaxInt64) / int64(time.Hour)
)

// parseTTLDuration parses a Go duration string extended with day (d, = 24h) and
// week (w, = 168h) units. Components in those units are summed here; all other
// components (h and below) are handed to time.ParseDuration unchanged, which
// keeps its exact integer handling for the common forms.
func parseTTLDuration(s string) (time.Duration, error) {
	// Fast path: no day/week unit means time.ParseDuration handles it all. No
	// other supported unit contains 'd' or 'w', so this check is exact.
	if !strings.ContainsAny(s, "dw") {
		return time.ParseDuration(s)
	}

	rest := s
	neg := false
	switch {
	case strings.HasPrefix(rest, "-"):
		neg, rest = true, rest[1:]
	case strings.HasPrefix(rest, "+"):
		rest = rest[1:]
	}

	var extra time.Duration      // accumulated day/week components
	var residual strings.Builder // remaining components, parsed by time.ParseDuration
	for rest != "" {
		m := ttlComponent.FindStringSubmatch(rest)
		if m == nil {
			return 0, fmt.Errorf("unrecognized duration component %q", rest)
		}
		switch m[2] {
		case "d", "w":
			hoursPerUnit := int64(hoursPerDay)
			if m[2] == "w" {
				hoursPerUnit = hoursPerWeek
			}
			if strings.Contains(m[1], ".") {
				f, err := strconv.ParseFloat(m[1], 64)
				if err != nil {
					return 0, err
				}
				ns := f * float64(hoursPerUnit) * float64(time.Hour)
				if ns > float64(math.MaxInt64) {
					return 0, fmt.Errorf("duration component %s%s overflows", m[1], m[2])
				}
				extra += time.Duration(ns)
			} else {
				n, err := strconv.ParseInt(m[1], 10, 64)
				if err != nil {
					return 0, err
				}
				if n > maxDurationHours/hoursPerUnit {
					return 0, fmt.Errorf("duration component %s%s overflows", m[1], m[2])
				}
				extra += time.Duration(n*hoursPerUnit) * time.Hour
			}
		default:
			residual.WriteString(m[1])
			residual.WriteString(m[2])
		}
		rest = rest[len(m[0]):]
	}

	total := extra
	if residual.Len() > 0 {
		d, err := time.ParseDuration(residual.String())
		if err != nil {
			return 0, err
		}
		total += d
	}
	if neg {
		total = -total
	}
	return total, nil
}

func init() {
	createRoleOverrides = append(createRoleOverrides, createRoleOverride)
	createBranchOverrides = append(createBranchOverrides, createBranchOverride)
}
