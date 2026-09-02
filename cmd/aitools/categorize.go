package aitools

import (
	"errors"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/telemetry/protos"
)

func classifyInstallError(err error) protos.AitoolsErrorCategory {
	if err == nil {
		return protos.AitoolsErrorCategoryUnspecified
	}

	if blocked, ok := errors.AsType[*installer.BlockedError](err); ok {
		return blockedErrorCategory(blocked)
	}
	if skill, ok := errors.AsType[*installer.SkillError](err); ok {
		return skillErrorCategory(skill)
	}
	return protos.AitoolsErrorCategoryUncategorized
}

func skillErrorCategory(e *installer.SkillError) protos.AitoolsErrorCategory {
	switch e.Reason {
	case installer.ReasonSkillNotFound:
		return protos.AitoolsErrorCategorySkillNotFound
	case installer.ReasonVersionIncompatible:
		return protos.AitoolsErrorCategoryVersionIncompatible
	default:
		return protos.AitoolsErrorCategoryUncategorized
	}
}

func blockedErrorCategory(e *installer.BlockedError) protos.AitoolsErrorCategory {
	switch e.Reason {
	case installer.ReasonCLINotOnPath:
		return protos.AitoolsErrorCategoryCLINotOnPath
	case installer.ReasonInstallFailed:
		return protos.AitoolsErrorCategoryPluginInstallFailed
	default:
		return protos.AitoolsErrorCategoryUncategorized
	}
}
