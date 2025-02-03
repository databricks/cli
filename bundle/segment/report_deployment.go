package segment

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/slack-go/slack"
)

type reportDeplyoment struct {
}

func ReportDeployment() *reportDeplyoment {
	return &reportDeplyoment{}
}

func (m *reportDeplyoment) Name() string {
	return "ReportDeployment"
}

func (m *reportDeplyoment) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {

	slackClient := slack.New(b.Config.Experimental.Segment.SlackToken)

	username := GetSlackUserFromEnv()

	deployMsg := fmt.Sprintf(
		"%s deployed bundle `%s` to `%s`",
		username,
		b.Config.Bundle.Name,
		b.Config.Bundle.Target,
	)

	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(
				slack.MarkdownType,
				deployMsg,
				false,
				false,
			),
			nil,
			nil,
		),
	}

	blocks = append(
		blocks,
		slack.NewContextBlock(
			"version",
			slack.NewTextBlockObject(
				slack.MarkdownType,
				fmt.Sprintf("Branch: <https://github.com/segmentio/profiles-data-lake-spark/tree/%s|%s>", b.Config.Variables["build_branch"].Value, b.Config.Variables["build_branch"].Value),
				false,
				false,
			),
			slack.NewTextBlockObject(
				slack.MarkdownType,
				fmt.Sprintf("Build Number: <https://buildkite.com/segment/profiles-data-lake-spark/builds/%s|%s>", b.Config.Variables["build_number"].Value, b.Config.Variables["build_number"].Value),
				false,
				false,
			),
			slack.NewTextBlockObject(
				slack.MarkdownType,
				fmt.Sprintf("Git SHA: <https://github.com/segmentio/profiles-data-lake-spark/commit/%s|%s>", b.Config.Variables["build_sha"].Value, b.Config.Variables["build_sha"].Value),
				false,
				false,
			),
		),
	)

	_, _, _, err := slackClient.SendMessage(
		"#eng-profiles-data-lake-deploys",
		slack.MsgOptionText(deployMsg, false),
		slack.MsgOptionBlocks(blocks...),
	)
	if err != nil {
		return diag.Errorf("failed to send slack message: %w", err)
	}

	fmt.Printf("Thank you for your input. Proceeding with deployment.\n\n")

	return nil
}
