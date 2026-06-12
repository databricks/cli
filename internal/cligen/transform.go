package main

import "github.com/databricks/cli/internal/clijson"

// fromContract converts the canonical commands block (decoded from cli.json into
// the verbatim contract types in internal/clijson) into the CLI's render model.
//
// clijson is the single source of truth for the wire shape; this transform is
// the one place that reads it, so a contract change surfaces here as a compile
// error rather than silent drift. Render-only state — the byID index, the
// ParentService/Subservices links, RequestBodyField pointer identity, and
// noPrompt — is left zero here and populated by CommandsBlock.Resolve().
func fromContract(c *clijson.CommandsBlock) *CommandsBlock {
	if c == nil {
		return nil
	}
	return &CommandsBlock{
		Services:            mapSlice(c.Services, fromService),
		WorkspaceDocsGroups: mapSlice(c.WorkspaceDocsGroups, fromDocsGroup),
		AccountDocsGroups:   mapSlice(c.AccountDocsGroups, fromDocsGroup),
	}
}

// mapSlice maps src through f, preserving nil so that omitempty fields stay
// absent rather than becoming an empty slice.
func mapSlice[S, D any](src []S, f func(S) D) []D {
	if src == nil {
		return nil
	}
	out := make([]D, len(src))
	for i, v := range src {
		out[i] = f(v)
	}
	return out
}

func fromDocsGroup(d *clijson.DocsGroupJSON) *DocsGroupJSON {
	if d == nil {
		return nil
	}
	return &DocsGroupJSON{Key: d.Key, DisplayName: d.DisplayName}
}

func fromPackageRef(p *clijson.PackageRef) *PackageRef {
	if p == nil {
		return nil
	}
	return &PackageRef{Name: p.Name}
}

func fromPascalRef(p *clijson.PascalRef) *PascalRef {
	if p == nil {
		return nil
	}
	return &PascalRef{PascalName: p.PascalName}
}

// fromNamedIdMap and fromFieldRef map the contract's PascalRef onto the render
// model's distinct-but-identically-shaped NamedIdMapJSON / FieldRef types.
func fromNamedIdMap(p *clijson.PascalRef) *NamedIdMapJSON {
	if p == nil {
		return nil
	}
	return &NamedIdMapJSON{PascalName: p.PascalName}
}

func fromFieldRef(p *clijson.PascalRef) *FieldRef {
	if p == nil {
		return nil
	}
	return &FieldRef{PascalName: p.PascalName}
}

func fromEnumEntry(e *clijson.EnumEntryJSON) *EnumEntryJSON {
	if e == nil {
		return nil
	}
	return &EnumEntryJSON{Content: e.Content}
}

func fromServiceListRef(l *clijson.ServiceListRefJSON) *ServiceListRefJSON {
	if l == nil {
		return nil
	}
	return &ServiceListRefJSON{
		NamedIdMap:           fromNamedIdMap(l.NamedIdMap),
		IsLegacyEmptyRequest: l.IsLegacyEmptyRequest,
		Request:              fromPascalRef(l.Request),
	}
}

func fromService(s *clijson.ServiceJSON) *ServiceJSON {
	if s == nil {
		return nil
	}
	return &ServiceJSON{
		ID:                    s.ID,
		Name:                  s.Name,
		Description:           s.Description,
		Package:               fromPackageRef(s.Package),
		IsAccounts:            s.IsAccounts,
		HasParent:             s.HasParent,
		IsDataPlane:           s.IsDataPlane,
		HasSubservices:        s.HasSubservices,
		IsHiddenCLI:           s.IsHiddenCLI,
		DocsGroup:             s.DocsGroup,
		LaunchStage:           s.LaunchStage,
		CLILaunchStageLabel:   s.CLILaunchStageLabel,
		CLILaunchStageBanner:  s.CLILaunchStageBanner,
		CLILaunchStageDisplay: s.CLILaunchStageDisplay,
		ParentServiceID:       s.ParentServiceID,
		SubserviceIDs:         s.SubserviceIDs,
		List:                  fromServiceListRef(s.List),
		Methods:               mapSlice(s.Methods, fromMethod),
	}
}

func fromMethod(m *clijson.MethodJSON) *MethodJSON {
	if m == nil {
		return nil
	}
	return &MethodJSON{
		Name:                           m.Name,
		Description:                    m.Description,
		Summary:                        m.Summary,
		Path:                           m.Path,
		IsLegacyEmptyRequest:           m.IsLegacyEmptyRequest,
		CanUseJson:                     m.CanUseJson,
		MustUseJson:                    m.MustUseJson,
		IsJsonOnly:                     m.IsJsonOnly,
		IsCrudCreate:                   m.IsCrudCreate,
		IsCrudRead:                     m.IsCrudRead,
		IsHiddenCLI:                    m.IsHiddenCLI,
		IsResponseByteStream:           m.IsResponseByteStream,
		HasRequiredPositionalArguments: m.HasRequiredPositionalArguments,
		LaunchStage:                    m.LaunchStage,
		CLILaunchStageLabel:            m.CLILaunchStageLabel,
		CLILaunchStageBanner:           m.CLILaunchStageBanner,
		CLILaunchStageDisplay:          m.CLILaunchStageDisplay,
		Request:                        fromEntity(m.Request),
		RequestBodyField:               fromField(m.RequestBodyField),
		Response:                       fromEntity(m.Response),
		ResponseBodyField:              fromPascalRef(m.ResponseBodyField),
		AllFields:                      mapSlice(m.AllFields, fromField),
		RequiredPositionalArguments:    mapSlice(m.RequiredPositionalArguments, fromField),
		Pagination:                     fromPagination(m.Pagination),
		Wait:                           fromWait(m.Wait),
		LongRunningOperation:           fromLRO(m.LongRunningOperation),
	}
}

func fromField(f *clijson.FieldJSON) *FieldJSON {
	if f == nil {
		return nil
	}
	return &FieldJSON{
		Name:               f.Name,
		Description:        f.Description,
		Required:           f.Required,
		IsPath:             f.IsPath,
		IsQuery:            f.IsQuery,
		IsComputed:         f.IsComputed,
		IsOutputOnly:       f.IsOutputOnly,
		IsRequestBodyField: f.IsRequestBodyField,
		IsOptionalObject:   f.IsOptionalObject,
		Entity:             fromEntity(f.Entity),
	}
}

func fromEntity(e *clijson.EntityJSON) *EntityJSON {
	if e == nil {
		return nil
	}
	return &EntityJSON{
		PascalName:                   e.PascalName,
		IsObject:                     e.IsObject,
		IsAny:                        e.IsAny,
		IsString:                     e.IsString,
		IsBool:                       e.IsBool,
		IsInt:                        e.IsInt,
		IsInt64:                      e.IsInt64,
		IsFloat64:                    e.IsFloat64,
		IsDuration:                   e.IsDuration,
		IsTimestamp:                  e.IsTimestamp,
		IsFieldMask:                  e.IsFieldMask,
		IsEmptyResponse:              e.IsEmptyResponse,
		ArrayValue:                   fromEntity(e.ArrayValue),
		MapValue:                     fromEntity(e.MapValue),
		Enum:                         mapSlice(e.Enum, fromEnumEntry),
		HasFieldMask:                 e.HasFieldMask,
		HasRequiredRequestBodyFields: e.HasRequiredRequestBodyFields,
		RequiredFields:               mapSlice(e.RequiredFields, fromField),
		RequiredInUrlFields:          mapSlice(e.RequiredInUrlFields, fromField),
		RequiredRequestBodyFields:    mapSlice(e.RequiredRequestBodyFields, fromField),
	}
}

func fromPagination(p *clijson.PaginationJSON) *PaginationJSON {
	if p == nil {
		return nil
	}
	return &PaginationJSON{
		Token:      fromToken(p.Token),
		Offset:     fromField(p.Offset),
		Limit:      fromField(p.Limit),
		MaxResults: fromField(p.MaxResults),
	}
}

func fromToken(t *clijson.TokenJSON) *TokenJSON {
	if t == nil {
		return nil
	}
	return &TokenJSON{PollField: fromField(t.PollField)}
}

func fromWait(w *clijson.WaitJSON) *WaitJSON {
	if w == nil {
		return nil
	}
	return &WaitJSON{
		Success:            mapSlice(w.Success, fromEnumEntry),
		Timeout:            w.Timeout,
		ComplexMessagePath: w.ComplexMessagePath,
		Poll:               fromPoll(w.Poll),
		StatusPath:         mapSlice(w.StatusPath, fromFieldRef),
		MessagePath:        mapSlice(w.MessagePath, fromFieldRef),
		MessagePathHead:    fromFieldRef(w.MessagePathHead),
	}
}

func fromPoll(p *clijson.PollJSON) *PollJSON {
	if p == nil {
		return nil
	}
	return &PollJSON{Response: fromPascalRef(p.Response)}
}

func fromLRO(l *clijson.LROJSON) *LROJSON {
	if l == nil {
		return nil
	}
	return &LROJSON{
		GetOperation: fromLROMethodRef(l.GetOperation),
		ResponseType: fromEntityRef(l.ResponseType),
	}
}

func fromLROMethodRef(r *clijson.LROMethodRef) *LROMethodRef {
	if r == nil {
		return nil
	}
	return &LROMethodRef{
		KebabName:  r.KebabName,
		PascalName: r.PascalName,
		Request:    fromPascalRef(r.Request),
	}
}

func fromEntityRef(e *clijson.EntityRef) *EntityRef {
	if e == nil {
		return nil
	}
	return &EntityRef{IsEmptyResponse: e.IsEmptyResponse}
}
