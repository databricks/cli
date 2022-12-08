package sdk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"gopkg.in/ini.v1"
)

type Profile struct {
	Name      string
	Host      string
	AccountID string
}

func (p Profile) Cloud() string {
	if strings.Contains(p.Host, ".azuredatabricks.net") {
		return "Azure"
	}
	if strings.Contains(p.Host, "gcp.databricks.com") {
		return "GCP"
	}
	return "AWS"
}

func loadProfiles() (profiles []Profile, err error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find homedir: %w", err)
	}
	file := filepath.Join(homedir, ".databrickscfg")
	iniFile, err := ini.Load(file)
	if err != nil {
		return
	}
	for _, v := range iniFile.Sections() {
		all := v.KeysHash()
		host, ok := all["host"]
		if !ok {
			// invalid profile
			continue
		}
		profiles = append(profiles, Profile{
			Name:      v.Name(),
			Host:      host,
			AccountID: all["account_id"],
		})
	}
	return profiles, nil
}

func askForWorkspaceProfile() (string, error) {
	profiles, err := loadProfiles()
	if err != nil {
		return "", err
	}
	var items []Profile
	for _, v := range profiles {
		if v.AccountID != "" {
			continue
		}
		items = append(items, v)
	}
	label := "~/.databrickscfg profile"
	i, _, err := (&promptui.Select{
		Label: label,
		Items: items,
		Templates: &promptui.SelectTemplates{
			Active:   `{{.Name | bold}} ({{.Host|faint}})`,
			Inactive: `{{.Name}}`,
			Selected: fmt.Sprintf(`{{ "%s" | faint }}: {{ .Name | bold }}`, label),
		},
		Stdin: os.Stdin,
	}).Run()
	if err != nil {
		return "", err
	}
	return items[i].Name, nil
}

func askForAccountProfile() (string, error) {
	profiles, err := loadProfiles()
	if err != nil {
		return "", err
	}
	var items []Profile
	for _, v := range profiles {
		if v.AccountID == "" {
			continue
		}
		items = append(items, v)
	}
	label := "~/.databrickscfg profile"
	i, _, err := (&promptui.Select{
		Label: label,
		Items: items,
		Templates: &promptui.SelectTemplates{
			Active:   `{{.Name | bold}} ({{.AccountID|faint}} {{.Cloud|faint}})`,
			Inactive: `{{.Name}}`,
			Selected: fmt.Sprintf(`{{ "%s" | faint }}: {{ .Name | bold }}`, label),
		},
		Stdin: os.Stdin,
	}).Run()
	if err != nil {
		return "", err
	}
	return items[i].Name, nil
}
