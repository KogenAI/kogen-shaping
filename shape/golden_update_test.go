package shape

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
)

var updateGoldens = flag.Bool("update", false, "update synthetic test goldens")

func writeGolden(t *testing.T, path string, value any, indent string) {
	t.Helper()
	var data []byte
	var err error
	if indent == "" {
		data, err = json.Marshal(value)
	} else {
		data, err = json.MarshalIndent(value, "", indent)
	}
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
