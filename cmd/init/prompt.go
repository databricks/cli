package init

import (
	"fmt"

	"github.com/databricks/bricks/project"
	"github.com/manifoldco/promptui"
)

type Results map[string]Answer

type Question interface {
	Ask(res Results) (key string, ans Answer, err error)
}

type Questions []Question

func (qq Questions) Ask(res Results) error {
	for _, v := range qq {
		key, ans, err := v.Ask(res)
		if err != nil {
			return err
		}
		res[key] = ans
	}
	return nil
}

type Text struct {
	key      string
	Label    string
	Default  func(res Results) string
	Callback AnswerCallback
}

func (t Text) Ask(res Results) (string, Answer, error) {
	def := ""
	if t.Default != nil {
		def = t.Default(res)
	}
	v, err := (&promptui.Prompt{
		Label:   t.Label,
		Default: def,
	}).Run()
	return t.key, Answer{
		Value:    v,
		Callback: t.Callback,
	}, err
}

type Choice struct {
	key     string
	Label   string
	Answers []Answer
}

func (q Choice) Ask(res Results) (string, Answer, error) {
	// TODO: validate and re-ask
	prompt := promptui.Select{
		Label: q.Label,
		Items: q.Answers,
		Templates: &promptui.SelectTemplates{
			Label:    `{{ .Value }}`,
			Details:  `{{ .Details | green }}`,
			Selected: fmt.Sprintf(`{{ "%s" | faint }}: {{ .Value | bold }}`, q.Label),
		},
	}
	i, _, err := prompt.Run()
	return q.key, q.Answers[i], err
}

type Answers []Answer

type AnswerCallback func(ans Answer, prj *project.Project, res Results)

type Answer struct {
	Value    string
	Details  string
	Callback AnswerCallback
}

func (a Answer) String() string {
	return a.Value
}
