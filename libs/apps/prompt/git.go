package prompt

import (
	"context"
	"strings"

	"github.com/charmbracelet/huh"
)

// Git repo onboarding choices returned by PromptCreateOrExistingRepo.
const (
	GitRepoCreate   = "create"
	GitRepoExisting = "existing"
)

// PromptDeployFromGit asks whether to deploy the app from a Git repository.
// Defaults to yes, since git-backed deployment is the recommended path.
func PromptDeployFromGit(ctx context.Context) (bool, error) {
	deploy := true
	err := huh.NewConfirm().
		Title("Deploy from a Git repository? (recommended)").
		Description("Deploy from a repo/ref instead of uploading local files — reproducible and rollback-able").
		Value(&deploy).
		WithTheme(AppkitTheme()).
		Run()
	if err != nil {
		return false, err
	}
	if deploy {
		printAnswered(ctx, "Git-backed deploy", "Yes")
	} else {
		printAnswered(ctx, "Git-backed deploy", "No")
	}
	return deploy, nil
}

// PromptCreateOrExistingRepo asks whether to create a new GitHub repo or use an
// existing one. Returns GitRepoCreate or GitRepoExisting.
func PromptCreateOrExistingRepo(ctx context.Context) (string, error) {
	choice := GitRepoCreate
	err := huh.NewSelect[string]().
		Title("Set up the Git repository").
		Options(
			huh.NewOption("Create a new GitHub repository", GitRepoCreate),
			huh.NewOption("Use an existing repository (clone into the current directory)", GitRepoExisting),
		).
		Value(&choice).
		WithTheme(AppkitTheme()).
		Run()
	if err != nil {
		return "", err
	}
	labels := map[string]string{GitRepoCreate: "Create new", GitRepoExisting: "Use existing"}
	printAnswered(ctx, "Repository", labels[choice])
	return choice, nil
}

// PromptRepoOwner asks which GitHub account the new repo should be created
// under: the personal account or an organization the user belongs to. When
// there are no orgs to choose from it returns the personal account without
// prompting.
func PromptRepoOwner(ctx context.Context, personal string, orgs []string) (string, error) {
	if len(orgs) == 0 {
		return personal, nil
	}
	options := make([]huh.Option[string], 0, len(orgs)+1)
	options = append(options, huh.NewOption(personal+" (personal)", personal))
	for _, o := range orgs {
		options = append(options, huh.NewOption(o, o))
	}
	owner := personal
	if err := huh.NewSelect[string]().
		Title("Repository owner").
		Description("your personal account or an organization you belong to").
		Options(options...).
		Value(&owner).
		WithTheme(AppkitTheme()).
		Run(); err != nil {
		return "", err
	}
	printAnswered(ctx, "Owner", owner)
	return owner, nil
}

// PromptRepoURL asks for a Git repository URL. An empty response is allowed and
// left for the caller to treat as "skip".
func PromptRepoURL(ctx context.Context, title, description string) (string, error) {
	var url string
	err := huh.NewInput().
		Title(title).
		Description(description).
		Placeholder("https://github.com/my-org/my-repo").
		Value(&url).
		WithTheme(AppkitTheme()).
		Run()
	if err != nil {
		return "", err
	}
	url = strings.TrimSpace(url)
	if url != "" {
		printAnswered(ctx, "Repository URL", url)
	}
	return url, nil
}

// PromptNewRepoDetails asks for the new repo/app name (defaulting to
// defaultName) and whether it should be private.
func PromptNewRepoDetails(ctx context.Context, defaultName string) (name string, private bool, err error) {
	name = defaultName
	if err = huh.NewInput().
		Title("Repository name").
		Description("also used as the app name — lowercase letters, numbers, hyphens (max 26 chars)").
		Placeholder("my-app").
		Value(&name).
		Validate(ValidateProjectName).
		WithTheme(AppkitTheme()).
		Run(); err != nil {
		return "", false, err
	}
	printAnswered(ctx, "Repository name", name)

	private = true
	if err = huh.NewConfirm().
		Title("Private repository?").
		Value(&private).
		WithTheme(AppkitTheme()).
		Run(); err != nil {
		return "", false, err
	}
	if private {
		printAnswered(ctx, "Visibility", "private")
	} else {
		printAnswered(ctx, "Visibility", "public")
	}
	return name, private, nil
}
