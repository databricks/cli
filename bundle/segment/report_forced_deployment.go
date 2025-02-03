package segment

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/slack-go/slack"
)

type reportForcedDeplyoment struct {
}

func ReportForcedDeployment() *reportForcedDeplyoment {
	return &reportForcedDeplyoment{}
}

func (m *reportForcedDeplyoment) Name() string {
	return "ReportForcedDeployment"
}

func (m *reportForcedDeplyoment) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {

	if os.Getenv("BUILDKITE") != "" {
		return nil
	}

	if b.Config.Bundle.Git.Branch == "" || b.Config.Bundle.Git.ActualBranch == "" {
		return nil
	}

	if b.Config.Bundle.Git.Branch == b.Config.Bundle.Git.ActualBranch || !b.Config.Bundle.Force {
		return nil
	}

	fmt.Printf(
		Red("It looks like you are using --force to deploy to a protected target (%s). \n")+
			"If you still want to proceed, please provide a justification. \n\n"+
			"If you want to abort the deployment, enter an empty justification.\n\n",
		b.Config.Bundle.Target,
	)

	reason, err := cmdio.Ask(ctx, "", "")
	if err != nil {
		return nil
	}

	if reason == "" {
		return diag.Errorf("Exiting due to user input.")
	}

	var message = fmt.Sprintf(
		"Test: %s is deploying bundle %s from branch %s to %s using --force. Justification: %s",
		GetSlackUserFromEnv(), b.Config.Bundle.Name, b.Config.Bundle.Git.ActualBranch, b.Config.Bundle.Target, reason,
	)

	slackClient := slack.New(b.Config.Experimental.Segment.SlackToken)

	_, _, _, err = slackClient.SendMessage(
		"#eng-profiles-data-lake-deploys",
		slack.MsgOptionText(fmt.Sprintf("This is a test: %s", message), false),
	)
	if err != nil {
		return diag.Errorf("failed to send slack message: %w", err)
	}

	fmt.Printf("Thank you for your input. Proceeding with deployment.\n\n")

	return nil
}

func Red(line string) string {
	return fmt.Sprintf("\033[91m%s\033[0m\n", line)
}
