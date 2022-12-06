package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/iancoleman/strcase"
)

type namedBlock struct {
	filePattern    string
	typeNamePrefix string
	name           string
	block          *tfjson.SchemaBlock
}

func (b *namedBlock) FieldName() string {
	return b.camelName()
}

func (b *namedBlock) TypeBase() []string {
	return []string{b.typeNamePrefix, b.camelName()}
}

func (b *namedBlock) TypeName() string {
	return strings.Join(b.TypeBase(), "")
}

func (b *namedBlock) TerraformName() string {
	return b.name
}

func (b *namedBlock) normalizedName() string {
	return normalizeName(b.name)
}

func (b *namedBlock) camelName() string {
	return strcase.ToCamel(b.normalizedName())
}

func (b *namedBlock) Generate(path string) error {
	w, err := walk(b.block, []string{b.typeNamePrefix, b.camelName()})
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(path, fmt.Sprintf(b.filePattern, b.normalizedName())))
	if err != nil {
		return err
	}

	defer f.Close()

	tmpl := template.Must(template.ParseFiles("./templates/block.go.tmpl"))
	return tmpl.Execute(f, w)
}
