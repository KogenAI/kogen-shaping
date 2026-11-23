// Command driver prints the progress lines DocumentationProgress gives for a row read from stdin.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/KogenAI/kogen-shaping/shape"
)

func main() {
	var row map[string]any
	if err := json.NewDecoder(os.Stdin).Decode(&row); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	started, finished := shape.DocumentationProgress(row)
	out, _ := json.Marshal(map[string]string{"started": started, "finished": finished})
	fmt.Println(string(out))
}
