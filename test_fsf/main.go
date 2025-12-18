package main

import (
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/api/ondemandscanning/v1"
)

func expect(title string, x any, result string) {
	b, err := json.Marshal(x)
	if err != nil {
		log.Fatal(err)
	}

	bb := string(b)
	if bb == result {
		fmt.Printf("OK %s %q\n", title, bb)
	} else {
		fmt.Printf("ERR %s Expected %q, got %q\n", title, result, bb)
	}
}

func main() {
	expect("Empty slice, no FSF", &ondemandscanning.GrafeasV1LayerDetails{
		BaseImages: []*ondemandscanning.GrafeasV1BaseImage{},
	}, "{}")

	expect("Empty slice, empty FSF", &ondemandscanning.GrafeasV1LayerDetails{
		BaseImages:      []*ondemandscanning.GrafeasV1BaseImage{},
		ForceSendFields: []string{""},
	}, "{}")

	expect("Empty slice + FSF", &ondemandscanning.GrafeasV1LayerDetails{
		BaseImages:      []*ondemandscanning.GrafeasV1BaseImage{},
		ForceSendFields: []string{"BaseImages"},
	}, `{"baseImages":[]}`)

	expect("non-empty slice + FSF", &ondemandscanning.GrafeasV1LayerDetails{
		BaseImages:      []*ondemandscanning.GrafeasV1BaseImage{{LayerCount: 25}},
		ForceSendFields: []string{"BaseImages"},
	}, `{"baseImages":[{"layerCount":25}]}`)

	expect("nil slice + FSF", &ondemandscanning.GrafeasV1LayerDetails{
		// BaseImages:      nil,
		ForceSendFields: []string{"BaseImages"},
	}, `{"baseImages":[]}`) // this is unexpected

	expect("nil slice + NullFields", &ondemandscanning.GrafeasV1LayerDetails{
		BaseImages: nil,
		NullFields: []string{"BaseImages"},
	}, `{"baseImages":null}`)

	expect("nil slice + FSF + NullFields", &ondemandscanning.GrafeasV1LayerDetails{
		BaseImages:      nil,
		ForceSendFields: []string{"BaseImages"},
		NullFields:      []string{"BaseImages"},
	}, `{"baseImages":null}`) // NullFields wins

	expect("Empty map, no FSF", &ondemandscanning.GrafeasV1SlsaProvenanceZeroTwoSlsaConfigSource{
		Digest: map[string]string{},
	}, "{}")

	expect("Empty map + FSF", &ondemandscanning.GrafeasV1SlsaProvenanceZeroTwoSlsaConfigSource{
		Digest:          map[string]string{},
		ForceSendFields: []string{"Digest"},
	}, `{"digest":{}}`)
}
