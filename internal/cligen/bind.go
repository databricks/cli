package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// MethodBind is the code-level binding of a command to the API: either an
// sdk-go client method with JSON mirror request/response types ("sdk"), or a
// plain REST call through libs/apiclient with untyped JSON ("raw") for
// endpoints sdk-go does not have.
type MethodBind struct {
	Kind string

	svc    *ServiceJSON
	method *MethodJSON
	sdk    *SdkMethod
	pkg    *SdkPackage
	// pathConsts are literal values for sdk-go path parameters (by position).
	pathConsts map[int]string

	// reqVar is the request variable, e.g. "getReq".
	reqVar string
	// bodyGo is the Go name of the request body field (RequestBodyField).
	bodyGo string
	// bodyType is the mirror type of the body field without pointer.
	bodyType string
	bodyPtr  bool

	fields map[string]*fieldBind
	// direct is set when the request has no mirror type (no JSON body) and
	// flags bind to the sdk-go request type itself.
	direct bool

	// Raw.
	Verb string

	// Legacy (hand-written client with the previous SDK's API).
	legacy  *legacyPkg
	legacyM *legacyMethod
}

type fieldBind struct {
	// expr is the Go expression of the field relative to the request
	// variable, e.g. ".Project.DisplayName". Empty when unmapped.
	expr string
	// kind is one of string, bool, int, int64, float64, enum, time, duration,
	// fieldmask, object, complex.
	kind string
	ptr  bool
	// enumType is the qualified mirror enum type.
	enumType string
	// raw: where the value goes ("path", "query", "body").
}

func (b *MethodBind) IsSdk() bool    { return b.Kind == "sdk" }
func (b *MethodBind) IsRaw() bool    { return b.Kind == "raw" }
func (b *MethodBind) IsLegacy() bool { return b.Kind == "legacy" }

func (b *MethodBind) LegacyCall() string {
	return b.client() + "." + b.legacy.clientField + "." + b.legacyM.name
}

func (b *MethodBind) LegacyKind() string { return b.legacyM.kind }

func (b *MethodBind) allFields() []*FieldJSON {
	all := append([]*FieldJSON{}, b.method.AllFields...)
	all = append(all, b.method.RequiredPositionalArguments...)
	if b.method.Request != nil {
		all = append(all, b.method.Request.RequiredFields...)
	}
	return all
}

func (b *MethodBind) client() string {
	if b.svc.IsAccounts {
		return "a"
	}
	return "w"
}

// ClientExpr is the sdk-go client, e.g. "w.Jobs".
func (b *MethodBind) ClientExpr() string { return b.client() + "." + b.pkg.ClientField }

func (b *MethodBind) SdkMethod() string { return b.sdk.Name }

// MirrorAlias is the import alias of the mirror package.
func (b *MethodBind) MirrorAlias() string {
	if b.pkg == nil {
		return ""
	}
	return b.pkg.MirrorAlias
}

// HasRequest reports whether the command has a request variable that flags,
// --json and overrides bind to.
func (b *MethodBind) HasRequest() bool {
	if b.method.IsLegacyEmptyRequest {
		return false
	}
	if b.IsRaw() {
		return true
	}
	if b.IsLegacy() {
		return b.legacyM.request != ""
	}
	return b.sdk.Request != ""
}

// ReqType is the request variable's type.
func (b *MethodBind) ReqType() string {
	if b.IsRaw() {
		return "map[string]any"
	}
	if b.IsLegacy() {
		return b.legacy.alias + "." + b.legacyM.request
	}
	if b.direct {
		return b.pkg.SdkAlias + "." + b.sdk.Request
	}
	return b.pkg.MirrorAlias + "." + b.sdk.Request
}

func (b *MethodBind) ReqVar() string { return b.reqVar }

// ReqInit initializes the request body object, if any.
func (b *MethodBind) ReqInit() string {
	if b.IsRaw() {
		return b.reqVar + " = map[string]any{}"
	}
	if b.bodyGo == "" {
		return ""
	}
	amp := ""
	if b.bodyPtr {
		amp = "&"
	}
	alias := b.MirrorAlias()
	if b.IsLegacy() {
		alias = b.legacy.alias
	}
	return fmt.Sprintf("%s.%s = %s%s.%s{}", b.reqVar, b.bodyGo, amp, alias, b.bodyType)
}

// JsonTarget is the value --json is unmarshalled into.
func (b *MethodBind) JsonTarget() string {
	if !b.IsRaw() && b.bodyGo != "" {
		return "&" + b.reqVar + "." + b.bodyGo
	}
	return "&" + b.reqVar
}

// SdkReq converts the mirror request into the sdk-go request and binds it to
// sdkReq; empty when the method has no request.
func (b *MethodBind) SdkReq() string {
	if b.sdk.Request == "" {
		return ""
	}
	if !b.HasRequest() {
		return fmt.Sprintf("sdkReq := &%s.%s{}\n", b.pkg.SdkAlias, b.sdk.Request) + b.pathConstsOn("sdkReq")
	}
	if b.direct {
		return fmt.Sprintf("sdkReq := &%s", b.reqVar)
	}
	return fmt.Sprintf(`sdkReq, err := %s.%sToSDK(&%s)
		if err != nil {
			return err
		}`, b.pkg.MirrorAlias, b.sdk.Request, b.reqVar)
}

// CallArgs are the arguments of the sdk-go call after ctx.
func (b *MethodBind) CallArgs() string {
	if b.sdk.Request == "" {
		return ""
	}
	return ", *sdkReq"
}

// SdkKind is "resp", "err", "iter", "waiter" or "op".
func (b *MethodBind) SdkKind() string {
	if b.method.Pagination != nil && b.sdk.Iter != "" {
		return "iter"
	}
	return b.sdk.Kind
}

func (b *MethodBind) IterMethod() string { return b.sdk.Iter }

func (b *MethodBind) HasResult() bool {
	return b.sdk.Result != "" && b.pkg.structs[b.sdk.Result] != nil
}

// FromSDK is the converter of the call result (response, wait result or list item).
func (b *MethodBind) FromSDK() string {
	t := b.sdk.Result
	if b.SdkKind() == "iter" {
		t = b.sdk.ItemType
	}
	return b.pkg.MirrorAlias + "." + t + "FromSDK"
}

// WaiterIDs renders the identifiers of a waiter, printed by --no-wait, as
// map entries keyed by their JSON names (e.g. "run_id": wait.GetRunId()).
func (b *MethodBind) WaiterIDs() string {
	var parts []string
	for _, g := range b.sdk.WaiterIDs {
		parts = append(parts, fmt.Sprintf("%q: wait.%s()", snakeName(strings.TrimPrefix(g, "Get")), g))
	}
	return strings.Join(parts, ", ")
}

// HasNoWaitCall reports whether --no-wait can send the request directly and
// render the API's immediate response, as the previous SDK's waiters exposed
// it (sdk-go waiters only keep the identifiers).
func (b *MethodBind) HasNoWaitCall() bool {
	if b.direct || b.sdk.baseResult == "" || b.pkg.structs[b.sdk.baseResult] == nil || b.sdk.Request == "" {
		return false
	}
	for _, f := range b.sdk.pathFields {
		if f == "" || strings.Contains(f, ".") {
			return false
		}
	}
	return true
}

// NoWaitCall sends the request of a waiter method through libs/apiclient and
// decodes the immediate response into `current`.
func (b *MethodBind) NoWaitCall() string {
	path := strings.ReplaceAll(b.sdk.path, "{}", "%s")
	var args []string
	for _, f := range b.sdk.pathFields {
		args = append(args, fmt.Sprintf("url.PathEscape(fmt.Sprint(utils.Deref(%s.%s)))", b.reqVar, f))
	}
	pathExpr := strconv.Quote(b.sdk.path)
	if len(args) > 0 {
		pathExpr = fmt.Sprintf("fmt.Sprintf(%q, %s)", path, strings.Join(args, ", "))
	}
	body := "nil"
	if b.sdk.hasBody {
		body = "&" + b.reqVar
		if b.sdk.bodyGo != "" {
			body = b.reqVar + "." + b.sdk.bodyGo
		}
	}
	query := "nil"
	if len(b.sdk.queryFields) > 0 {
		var names []string
		for name := range b.sdk.queryFields {
			names = append(names, name)
		}
		sort.Strings(names)
		var parts []string
		for _, name := range names {
			parts = append(parts, fmt.Sprintf("%q: %s.%s", name, b.reqVar, b.sdk.queryFields[name]))
		}
		query = "map[string]any{" + strings.Join(parts, ", ") + "}"
	}
	return fmt.Sprintf(`api, err := apiclient.New(%s.Config)
			if err != nil {
				return err
			}
			var current %s.%s
			err = api.Do(ctx, %q, %s, %s, %s, %s, &current)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, current)`, b.client(), b.pkg.MirrorAlias, b.sdk.baseResult, b.sdk.verb, pathExpr, b.RawHeaders(), query, body)
}

// ListItems is the Go field holding the items of a list response, for list
// methods that sdk-go does not page through (no ...Iter method). The previous
// SDK returned these items as a list, which is what the commands render.
func (b *MethodBind) ListItems() string {
	if b.method.Pagination == nil || b.sdk.Kind != "resp" || b.sdk.Iter != "" {
		return ""
	}
	st := b.pkg.structs[b.sdk.Result]
	if st == nil {
		return ""
	}
	field := ""
	for _, f := range st.Fields {
		if strings.HasPrefix(f.Type, "[]") && f.Type != "[]byte" {
			if field != "" {
				return ""
			}
			field = f.GoName
		}
	}
	return field
}

// PollCall returns the call fetching the current state of a waiter's
// resource (used by --no-wait), or "" if the poll method is unknown.
func (b *MethodBind) PollCall() string {
	pm := b.pollMethod()
	if pm == nil {
		return ""
	}
	var fields []string
	for _, g := range b.sdk.WaiterIDs {
		fields = append(fields, fmt.Sprintf("%s: new(wait.%s())", strings.TrimPrefix(g, "Get"), g))
	}
	return fmt.Sprintf("%s.%s(ctx, %s.%s{%s})", b.ClientExpr(), pm.Name, b.pkg.SdkAlias, pm.Request, strings.Join(fields, ", "))
}

func (b *MethodBind) PollFromSDK() string {
	return b.pkg.MirrorAlias + "." + b.pollMethod().Result + "FromSDK"
}

func (b *MethodBind) pollMethod() *SdkMethod {
	if b.sdk.pollRequest == "" {
		return nil
	}
	var names []string
	for name := range b.pkg.methods {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		m := b.pkg.methods[name]
		if m.Request == b.sdk.pollRequest && m.Kind == "resp" && m.Result == b.sdk.Result && b.pkg.structs[m.Result] != nil {
			return m
		}
	}
	return nil
}

// GetOperation returns the sdk-go method polling the operation of an LRO, if
// there is one with the name cli.json uses.
func (b *MethodBind) GetOperation() string {
	lro := b.method.LongRunningOperation
	if lro == nil || lro.GetOperation == nil {
		return ""
	}
	if m := b.pkg.methods[lro.GetOperation.PascalName]; m != nil && m.Request != "" && m.Kind == "resp" && b.pkg.structs[m.Result] != nil {
		return m.Name
	}
	return ""
}

func (b *MethodBind) GetOperationRequest() string {
	return b.pkg.SdkAlias + "." + b.pkg.methods[b.GetOperation()].Request
}

func (b *MethodBind) GetOperationFromSDK() string {
	return b.pkg.MirrorAlias + "." + b.pkg.methods[b.GetOperation()].Result + "FromSDK"
}

// field returns the binding of a cli.json field.
func (b *MethodBind) field(f *FieldJSON) *fieldBind {
	if fb, ok := b.fields[f.Name]; ok {
		return fb
	}
	return &fieldBind{}
}

// Mapped reports whether the field has a Go destination and a flag-able kind.
func (b *MethodBind) Mapped(f *FieldJSON) bool {
	fb := b.field(f)
	if b.IsRaw() {
		return true
	}
	return fb.expr != ""
}

func (b *MethodBind) target(f *FieldJSON) string {
	return b.reqVar + b.field(f).expr
}

func rawVar(f *FieldJSON, m *MethodJSON) string {
	return m.CamelName() + f.PascalName() + "Raw"
}

func flagFunc(kind string, ptr bool) string {
	name := map[string]string{"string": "String", "bool": "Bool", "int": "Int", "int64": "Int64", "float64": "Float64", "float32": "Float32"}[kind]
	if ptr {
		return "flags." + name + "PtrVar"
	}
	return name + "Var"
}

func rawKind(f *FieldJSON) string {
	e := f.Entity
	switch {
	case e == nil:
		return "complex"
	case e.IsString:
		return "string"
	case e.IsBool:
		return "bool"
	case e.IsInt64:
		return "int64"
	case e.IsInt:
		return "int"
	case e.IsFloat64:
		return "float64"
	case len(e.Enum) > 0:
		return "enum"
	case e.IsTimestamp:
		return "time"
	case e.IsDuration:
		return "duration"
	case e.IsFieldMask:
		return "fieldmask"
	case e.IsObject:
		return "object"
	}
	return "complex"
}

func enumValues(f *FieldJSON) string {
	var vs []string
	for _, e := range f.Entity.Enum {
		vs = append(vs, strconv.Quote(e.Content))
	}
	return strings.Join(vs, ", ")
}

// RawDecls declares the variables raw flags are bound to.
func (b *MethodBind) RawDecls() string {
	if !b.IsRaw() {
		return ""
	}
	var sb strings.Builder
	seen := map[string]bool{}
	for _, f := range b.method.AllFields {
		if seen[f.Name] {
			continue
		}
		seen[f.Name] = true
		k := rawKind(f)
		goType := map[string]string{"string": "string", "bool": "bool", "int": "int", "int64": "int64", "float64": "float64", "enum": "string"}[k]
		if goType == "" {
			continue
		}
		fmt.Fprintf(&sb, "var %s %s\n", rawVar(f, b.method), goType)
	}
	return sb.String()
}

// FlagDef returns the statement registering a flag for field f.
func (b *MethodBind) FlagDef(f *FieldJSON, name, usage string) string {
	usage = "`" + usage + "`"
	if b.IsRaw() {
		k := rawKind(f)
		v := rawVar(f, b.method)
		switch k {
		case "string", "bool", "int", "int64", "float64":
			return fmt.Sprintf("cmd.Flags().%s(&%s, %q, %s, %s)", flagFunc(k, false), v, name, v, usage)
		case "enum":
			return fmt.Sprintf("flags.EnumVar(cmd.Flags(), &%s, %q, %s, %q, %s)", v, name, usage, f.Entity.PascalName, enumValues(f))
		}
		return "// TODO: complex arg: " + f.Name
	}
	fb := b.field(f)
	if fb.expr == "" {
		return "// TODO: field not in sdk-go request: " + f.Name
	}
	t := b.target(f)
	switch fb.kind {
	case "string", "bool", "int", "int64", "float64", "float32":
		if fb.kind == "string" && f.Entity != nil && len(f.Entity.Enum) > 0 && fb.ptr {
			return fmt.Sprintf("flags.EnumPtrVar(cmd.Flags(), &%s, %q, %s, %q, %s)", t, name, usage, f.Entity.PascalName, enumValues(f))
		}
		if fb.ptr {
			return fmt.Sprintf("%s(cmd.Flags(), &%s, %q, %s)", flagFunc(fb.kind, true), t, name, usage)
		}
		return fmt.Sprintf("cmd.Flags().%s(&%s, %q, %s, %s)", flagFunc(fb.kind, false), t, name, t, usage)
	case "legacyenum":
		return fmt.Sprintf("cmd.Flags().Var(&%s, %q, %s)", t, name, usage)
	case "bytes":
		return fmt.Sprintf("flags.Base64Var(cmd.Flags(), &%s, %q, %s)", t, name, usage)
	case "time", "duration":
		// cli.json describes the field as a plain string; parse it the way the
		// API's JSON encodes it.
		return fmt.Sprintf("flags.JSONStringPtrVar(cmd.Flags(), &%s, %q, %s)", t, name, usage)
	case "enum":
		typ := f.Entity.PascalName
		if typ == "" {
			typ = fb.enumType[strings.Index(fb.enumType, ".")+1:]
		}
		vals := ""
		if f.Entity != nil && len(f.Entity.Enum) > 0 {
			vals = ", " + enumValues(f)
		}
		if fb.ptr {
			return fmt.Sprintf("flags.EnumPtrVar(cmd.Flags(), &%s, %q, %s, %q%s)", t, name, usage, typ, vals)
		}
		return fmt.Sprintf("flags.EnumVar(cmd.Flags(), &%s, %q, %s, %q%s)", t, name, usage, typ, vals)
	}
	return "// TODO: complex arg: " + f.Name
}

// ArgScan assigns positional argument src (e.g. "args[0]") to field f.
func (b *MethodBind) ArgScan(f *FieldJSON, src string) string {
	if b.IsRaw() {
		v := rawVar(f, b.method)
		switch k := rawKind(f); k {
		case "string", "enum":
			return fmt.Sprintf("%s = %s", v, src)
		case "bool", "int", "int64", "float64":
			return fmt.Sprintf(`_, err = fmt.Sscan(%s, &%s)
	if err != nil {
		return fmt.Errorf("invalid %s: %%s", %s)
	}`, src, v, f.ConstantName(), src)
		case "time", "duration", "fieldmask":
			return b.WellKnown(f, src)
		}
		return fmt.Sprintf("%s[%q] = %s", b.reqVar, f.Name, src)
	}
	fb := b.field(f)
	if fb.expr == "" {
		return "_ = " + src + " // TODO: field not in sdk-go request: " + f.Name
	}
	t := b.target(f)
	switch fb.kind {
	case "string":
		if fb.ptr {
			return fmt.Sprintf("%s = new(%s)", t, src)
		}
		return fmt.Sprintf("%s = %s", t, src)
	case "enum":
		if fb.ptr {
			return fmt.Sprintf("%s = new(%s(%s))", t, fb.enumType, src)
		}
		return fmt.Sprintf("%s = %s(%s)", t, fb.enumType, src)
	case "legacyenum":
		return fmt.Sprintf(`_, err = fmt.Sscan(%s, &%s)
	if err != nil {
		return fmt.Errorf("invalid %s: %%s", %s)
	}`, src, t, f.ConstantName(), src)
	case "bool", "int", "int64", "float64":
		v := b.method.CamelName() + f.PascalName() + "Arg"
		assign := v
		if fb.ptr {
			assign = "&" + v
		}
		return fmt.Sprintf(`var %s %s
	_, err = fmt.Sscan(%s, &%s)
	if err != nil {
		return fmt.Errorf("invalid %s: %%s", %s)
	}
	%s = %s`, v, fb.kind, src, v, f.ConstantName(), src, t, assign)
	case "time", "duration", "fieldmask":
		return b.WellKnown(f, src)
	}
	return "_ = " + src + " // TODO: complex arg: " + f.Name
}

// WellKnown parses a timestamp, duration or field mask string src into f.
func (b *MethodBind) WellKnown(f *FieldJSON, src string) string {
	if b.IsRaw() {
		return fmt.Sprintf("%s[%q] = %s", b.reqVar, f.Name, src)
	}
	fb := b.field(f)
	if fb.expr == "" {
		return "_ = " + src
	}
	t := b.target(f)
	switch fb.kind {
	case "fieldmask", "string":
		if fb.ptr {
			return fmt.Sprintf("%s = new(%s)", t, src)
		}
		return fmt.Sprintf("%s = %s", t, src)
	case "time", "duration":
		typ := "sdkcoretypes.Time"
		if fb.kind == "duration" {
			typ = "sdkcoretypes.Duration"
		}
		v := f.CamelName() + "Field"
		amp := "&"
		if !fb.ptr {
			amp = ""
		}
		return fmt.Sprintf(`var %s %s
		err = json.Unmarshal([]byte(strconv.Quote(%s)), &%s)
		if err != nil {
			return fmt.Errorf("invalid %s: %%s", %s)
		}
		%s = %s%s`, v, typ, src, v, f.ConstantName(), src, t, amp, v)
	}
	return "_ = " + src
}

// RawAssemble moves flag values that were set into the raw request and
// computes the path and query parameters.
func (b *MethodBind) RawAssemble() string {
	var sb strings.Builder
	if !b.HasRequest() {
		fmt.Fprintf(&sb, "%s := map[string]any{}\n", b.reqVar)
	}
	sb.WriteString("rawQuery := map[string]any{}\n")
	for _, f := range b.method.AllFields {
		k := rawKind(f)
		v := rawVar(f, b.method)
		loc := "body"
		if f.IsQuery {
			loc = "query"
		}
		if f.IsPath {
			continue
		}
		switch k {
		case "string", "bool", "int", "int64", "float64", "enum":
			dst := b.reqVar
			if loc == "query" {
				dst = "rawQuery"
			}
			fmt.Fprintf(&sb, "if cmd.Flags().Changed(%q) || %s != %s {\n%s[%q] = %s\n}\n", f.KebabName(), v, zeroOf(k), dst, f.Name, v)
		}
	}
	// Path.
	params := pathParams(b.method.Path)
	byName := map[string]*FieldJSON{}
	for _, f := range b.method.AllFields {
		byName[f.Name] = f
	}
	path := pathParamRe.ReplaceAllString(b.method.Path, "%s")
	var args []string
	for _, p := range params {
		if p == "account_id" {
			args = append(args, b.client()+".Config.AccountID")
			continue
		}
		f := byName[p]
		if f == nil {
			args = append(args, `""`)
			continue
		}
		args = append(args, fmt.Sprintf("fmt.Sprint(%s)", rawVar(f, b.method)))
	}
	if len(args) == 0 {
		fmt.Fprintf(&sb, "rawPath := %q\n", b.method.Path)
	} else {
		fmt.Fprintf(&sb, "rawPath := fmt.Sprintf(%q, %s)\n", path, strings.Join(args, ", "))
	}
	return sb.String()
}

func zeroOf(k string) string {
	switch k {
	case "bool":
		return "false"
	case "string", "enum":
		return `""`
	}
	return "0"
}

// RawBody is the request body argument of the raw call.
func (b *MethodBind) RawBody() string {
	switch b.Verb {
	case "GET", "DELETE", "HEAD":
		return "nil"
	}
	return b.reqVar
}

// RawQuery merges body fields into the query for verbs without a body.
func (b *MethodBind) RawQuery() string {
	switch b.Verb {
	case "GET", "DELETE", "HEAD":
		return "maps.Copy(rawQuery, " + b.reqVar + ")"
	}
	return ""
}

func (b *MethodBind) RawHeaders() string {
	if b.svc.IsAccounts {
		return "nil"
	}
	return "auth.WorkspaceIDHeaders(w.Config)"
}

// bindAll computes the binding of every method.
func bindAll(batch *CommandsBlock, idx *sdkIndex, legacy map[string]*legacyPkg) error {
	for _, s := range batch.Services {
		for _, m := range s.Methods {
			b := &MethodBind{svc: s, method: m, reqVar: m.CamelName() + "Req", fields: map[string]*fieldBind{}}
			m.Bind = b
			if lp := legacy[s.Name]; lp != nil && b.bindLegacy(lp) {
				continue
			}
			sm, consts := idx.match(s, m)
			if sm == nil {
				b.Kind = "raw"
				b.Verb = methodVerb(s, m)
				if b.Verb == "" {
					if strings.HasPrefix(m.Name, "List") || strings.HasPrefix(m.Name, "Get") {
						b.Verb = "GET"
					} else {
						return fmt.Errorf("%s.%s: no sdk-go method and no HTTP verb", s.Name, m.Name)
					}
				}
				debugf("RAW %s.%s %s %s\n", s.Name, m.Name, b.Verb, m.Path)
				continue
			}
			b.Kind = "sdk"
			b.sdk = sm
			b.pkg = sm.Pkg
			b.pathConsts = consts
			b.bindFields()
		}
	}
	return nil
}

func (b *MethodBind) bindFields() {
	req := b.pkg.structs[b.sdk.Request]
	if req == nil {
		model := b.pkg.models[b.sdk.Request]
		if model == nil {
			return
		}
		b.direct = true
		req = &MirrorStruct{Fields: model.Fields, byJSON: map[string]*MirrorField{}}
		for name, goName := range b.sdk.queryFields {
			for _, f := range model.Fields {
				if f.GoName == goName {
					req.byJSON[name] = f
				}
			}
		}
	}
	var body *MirrorStruct
	if rb := b.method.RequestBodyField; rb != nil {
		if bf := req.byJSON[rb.Name]; bf != nil {
			b.bodyGo = bf.GoName
			b.bodyPtr = strings.HasPrefix(bf.Type, "*")
			b.bodyType = strings.TrimPrefix(bf.Type, "*")
			body = b.pkg.structs[b.bodyType]
		}
	}
	// Path parameters by position.
	cliParams := pathParams(b.method.Path)
	pathGo := map[string]string{}
	// sdk-go path parameters line up with cli.json ones, except that a
	// literal segment of the cli.json path may fill an sdk-go parameter
	// (pathConsts) and the account ID is implicit in sdk-go (field "").
	j := 0
	for i, sf := range b.sdk.pathFields {
		if _, isConst := b.pathConsts[i]; isConst {
			continue
		}
		for j < len(cliParams) && cliParams[j] == "account_id" && sf != "" {
			j++
		}
		if j >= len(cliParams) {
			break
		}
		if sf != "" {
			pathGo[cliParams[j]] = sf
		}
		j++
	}
	for _, f := range b.allFields() {
		if _, ok := b.fields[f.Name]; ok {
			continue
		}
		fb := &fieldBind{}
		b.fields[f.Name] = fb
		var mf *MirrorField
		var prefix string
		if goPath, ok := pathGo[f.Name]; ok && f.IsPath {
			mf, prefix = b.resolveGoPath(req, goPath)
		}
		if mf == nil && body != nil && !f.IsPath && !f.IsQuery {
			if mf = body.byJSON[f.Name]; mf != nil {
				prefix = "." + b.bodyGo
			}
		}
		if mf == nil {
			mf = req.byJSON[f.Name]
		}
		if mf == nil && body != nil {
			if mf = body.byJSON[f.Name]; mf != nil {
				prefix = "." + b.bodyGo
			}
		}
		if mf == nil {
			debugf("UNMAPPED %s.%s field %s\n", b.svc.Name, b.method.Name, f.Name)
			continue
		}
		fb.expr = prefix + "." + mf.GoName
		fb.ptr = strings.HasPrefix(mf.Type, "*")
		base := strings.TrimPrefix(mf.Type, "*")
		switch {
		case base == "string" || base == "bool" || base == "int" || base == "int64" || base == "float64" || base == "float32":
			fb.kind = base
			if f.Entity != nil && f.Entity.IsFieldMask {
				fb.kind = "fieldmask"
			}
		case base == "[]byte":
			fb.kind = "bytes"
		case base == "types.Time":
			fb.kind = "time"
		case base == "types.Duration":
			fb.kind = "duration"
		case strings.HasPrefix(base, "[]") || strings.HasPrefix(base, "map[") || strings.Contains(base, "."):
			fb.kind = "complex"
		case b.pkg.structs[base] != nil:
			fb.kind = "object"
		default:
			fb.kind = "enum"
			fb.enumType = b.pkg.MirrorAlias + "." + base
			if b.direct {
				fb.enumType = b.pkg.SdkAlias + "." + base
			}
		}
	}
}

// resolveGoPath follows a dotted Go field path ("Project.Name") from the
// request struct, returning the leaf field and the expression prefix.
func (b *MethodBind) resolveGoPath(req *MirrorStruct, goPath string) (*MirrorField, string) {
	parts := strings.Split(goPath, ".")
	cur := req
	prefix := ""
	for i, p := range parts {
		var found *MirrorField
		for _, f := range cur.Fields {
			if f.GoName == p {
				found = f
			}
		}
		if found == nil {
			return nil, ""
		}
		if i == len(parts)-1 {
			return found, prefix
		}
		prefix += "." + p
		cur = b.pkg.structs[strings.TrimPrefix(found.Type, "*")]
		if cur == nil {
			return nil, ""
		}
	}
	return nil, ""
}

// PathConsts assigns literal path parameter values (e.g. the object type of
// the typed permission endpoints) to the request.
func (b *MethodBind) PathConsts() string {
	if !b.HasRequest() {
		return ""
	}
	return b.pathConstsOn(b.reqVar)
}

func (b *MethodBind) pathConstsOn(v string) string {
	if len(b.pathConsts) == 0 {
		return ""
	}
	var keys []int
	for k := range b.pathConsts {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var sb strings.Builder
	for _, k := range keys {
		if k >= len(b.sdk.pathFields) || b.sdk.pathFields[k] == "" {
			continue
		}
		fmt.Fprintf(&sb, "%s.%s = new(%q)\n", v, b.sdk.pathFields[k], b.pathConsts[k])
	}
	return sb.String()
}

// NamedIdMap returns the expression collecting the name to id map for the
// interactive prompt of service s, or "" if it cannot be derived.
func (s *ServiceJSON) NamedIdMapCall() string {
	if s.List == nil || s.List.NamedIdMap == nil {
		return ""
	}
	var list *MethodJSON
	for _, m := range s.Methods {
		if m.Name == "List" {
			list = m
		}
	}
	if list == nil || list.Bind == nil || !list.Bind.IsSdk() || list.Bind.sdk.Iter == "" {
		return ""
	}
	b := list.Bind
	item := b.pkg.structs[b.sdk.ItemType]
	if item == nil {
		return ""
	}
	name := strings.TrimSuffix(s.List.NamedIdMap.PascalName, "Map")
	i := strings.LastIndex(name, "To")
	if i < 0 {
		return ""
	}
	left, right := name[:i], name[i+2:]
	namePath := b.findPath(item, left, true)
	idPath := b.findPath(item, right, false)
	if namePath == nil || idPath == nil {
		return ""
	}
	req := ""
	if b.sdk.Request != "" {
		req = ", " + b.pkg.SdkAlias + "." + b.sdk.Request + "{}"
	}
	q := func(p []string) string {
		var out []string
		for _, x := range p {
			out = append(out, strconv.Quote(x))
		}
		return "[]string{" + strings.Join(out, ", ") + "}"
	}
	return fmt.Sprintf("cmdgen.NamedIdMap(ctx, %s.%s.%s(ctx%s), %s, %s)", b.client(), b.pkg.ClientField, b.sdk.Iter, req, q(namePath), q(idPath))
}

// findPath finds a Go field path in st whose concatenated names equal s
// (exactly, or as a suffix when allowPrefix is set, to skip the old type name).
func (b *MethodBind) findPath(st *MirrorStruct, s string, allowPrefix bool) []string {
	for start := 0; start < len(s); start++ {
		if start > 0 && !allowPrefix {
			break
		}
		if s[start] < 'A' || s[start] > 'Z' {
			continue
		}
		if p := b.walkPath(st, s[start:], 0); p != nil {
			return p
		}
	}
	return nil
}

func (b *MethodBind) walkPath(st *MirrorStruct, s string, depth int) []string {
	if st == nil || depth > 2 {
		return nil
	}
	for _, f := range st.Fields {
		if !strings.HasPrefix(s, f.GoName) {
			continue
		}
		rest := s[len(f.GoName):]
		if rest == "" {
			return []string{f.GoName}
		}
		if p := b.walkPath(b.pkg.structs[strings.TrimPrefix(f.Type, "*")], rest, depth+1); p != nil {
			return append([]string{f.GoName}, p...)
		}
	}
	return nil
}

// Imports of a service file: mirror and sdk-go packages used by its methods.
func (s *ServiceJSON) BindImports() []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range s.Methods {
		if m.Bind != nil && m.Bind.IsLegacy() {
			imp := fmt.Sprintf("%s %q", m.Bind.legacy.alias, m.Bind.legacy.importPath)
			if !seen[imp] {
				seen[imp] = true
				out = append(out, imp)
			}
		}
		if m.Bind == nil || !m.Bind.IsSdk() {
			continue
		}
		p := m.Bind.pkg
		for _, imp := range []string{
			fmt.Sprintf("%s %q", p.MirrorAlias, p.MirrorPath),
			fmt.Sprintf("%s %q", p.SdkAlias, p.ImportPath),
		} {
			if !seen[imp] {
				seen[imp] = true
				out = append(out, imp)
			}
		}
	}
	sort.Strings(out)
	return out
}

// HasWaitResult is false for waits and calls that produce nothing.
func (b *MethodBind) HasWaitResult() bool {
	switch b.sdk.Kind {
	case "waiter", "op":
		return b.sdk.Result != ""
	}
	return true
}
