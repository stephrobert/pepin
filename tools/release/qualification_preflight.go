package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/stephrobert/pepin/internal/quality"
)

func checkQualifications(root string, now time.Time) error {
	records, err := quality.LoadQualifications(root)
	if err != nil {
		return err
	}
	providers := map[string]bool{}
	for _, q := range records {
		if _, exists := providers[q.Provider]; !exists {
			providers[q.Provider] = false
		}
		if q.FreshAt(now) {
			providers[q.Provider] = true
		}
	}
	var missing []string
	for provider, fresh := range providers {
		if !fresh {
			missing = append(missing, provider)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("aucune qualification fraîche pour %s ; refaire une qualification live",
			strings.Join(missing, ", "))
	}
	return nil
}

func main() {
	if err := checkQualifications(".", time.Now().UTC()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("les qualifications authentifiées enregistrées sont fraîches par fournisseur")
}
