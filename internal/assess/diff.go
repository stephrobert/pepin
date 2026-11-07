package assess

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/scankit/assessment"
)

// BundleDiff compares two internally verified evidence bundles.
type BundleDiff struct {
	Before            BundleInfo       `json:"before"`
	After             BundleInfo       `json:"after"`
	VersionChanges    []VersionChange  `json:"version_changes,omitempty"`
	ResultChanges     []ResultChange   `json:"result_changes,omitempty"`
	ResourceChanges   []ResourceChange `json:"resource_changes,omitempty"`
	ResourcesCompared bool             `json:"resources_compared"`
	Summary           DiffSummary      `json:"summary"`
}

// BundleInfo is the provenance needed to interpret one side of a bundle diff.
type BundleInfo struct {
	Generated       string               `json:"generated,omitempty"`
	Tool            assessment.Component `json:"tool"`
	Ruleset         assessment.Component `json:"ruleset"`
	Target          assessment.Target    `json:"target"`
	Source          string               `json:"source,omitempty"`
	Format          string               `json:"bundle_format"`
	InventorySchema string               `json:"inventory_schema,omitempty"`
}

// VersionChange identifies provenance or scope metadata that differs between runs.
type VersionChange struct {
	Component string `json:"component"`
	Before    string `json:"before"`
	After     string `json:"after"`
}

// ResultChange is a status change for a control and subject.
type ResultChange struct {
	Control string            `json:"control"`
	Subject string            `json:"subject,omitempty"`
	Kind    string            `json:"kind"`
	Before  assessment.Status `json:"before,omitempty"`
	After   assessment.Status `json:"after,omitempty"`
}

// ResourceChange records a resource added to or removed from the inventory.
type ResourceChange struct {
	Provider string `json:"provider"`
	Type     string `json:"type"`
	ID       string `json:"id"`
	Kind     string `json:"kind"`
}

// DiffSummary counts the changes without treating loss of visibility as a fix.
type DiffSummary struct {
	Fixed              int `json:"fixed"`
	Regressed          int `json:"regressed"`
	VisibilityLost     int `json:"visibility_lost"`
	VisibilityRestored int `json:"visibility_restored"`
	StatusChanged      int `json:"status_changed"`
	ResultsAdded       int `json:"results_added"`
	ResultsRemoved     int `json:"results_removed"`
	ResourcesAdded     int `json:"resources_added"`
	ResourcesRemoved   int `json:"resources_removed"`
}

type diffBundle struct {
	info       BundleInfo
	assessment assessment.Assessment
	resources  []model.Resource
	hasInput   bool
}

type resultKey struct {
	control string
	subject string
}

type resourceKey struct {
	provider string
	typeName string
	id       string
}

// DiffBundles verifies each bundle's internal checksums before comparing the
// stable control/status identifiers and resource identities. It does not verify
// a detached signature; use `pepin verify --pubkey` when signer authenticity is
// required.
func DiffBundles(beforeDir, afterDir string) (BundleDiff, error) {
	before, err := readDiffBundle(beforeDir)
	if err != nil {
		return BundleDiff{}, fmt.Errorf("before bundle: %w", err)
	}
	after, err := readDiffBundle(afterDir)
	if err != nil {
		return BundleDiff{}, fmt.Errorf("after bundle: %w", err)
	}
	if before.info.Target.Provider != after.info.Target.Provider {
		return BundleDiff{}, fmt.Errorf(i18n.T(
			"fournisseurs différents : %q et %q",
			"providers differ: %q and %q"), before.info.Target.Provider, after.info.Target.Provider)
	}
	if before.info.Target.ID != "" && after.info.Target.ID != "" &&
		before.info.Target.ID != after.info.Target.ID {
		return BundleDiff{}, fmt.Errorf(i18n.T(
			"cibles différentes : %q et %q",
			"targets differ: %q and %q"), before.info.Target.ID, after.info.Target.ID)
	}

	resultChanges, summary, err := compareResults(before.assessment.Results, after.assessment.Results)
	if err != nil {
		return BundleDiff{}, err
	}
	var resourceChanges []ResourceChange
	resourcesCompared := before.hasInput && after.hasInput
	if resourcesCompared {
		resourceChanges, err = compareResources(before.resources, after.resources)
		if err != nil {
			return BundleDiff{}, err
		}
		for _, change := range resourceChanges {
			switch change.Kind {
			case "added":
				summary.ResourcesAdded++
			case "removed":
				summary.ResourcesRemoved++
			}
		}
	}

	return BundleDiff{
		Before:            before.info,
		After:             after.info,
		VersionChanges:    compareBundleInfo(before.info, after.info),
		ResultChanges:     resultChanges,
		ResourceChanges:   resourceChanges,
		ResourcesCompared: resourcesCompared,
		Summary:           summary,
	}, nil
}

func readDiffBundle(dir string) (diffBundle, error) {
	if err := VerifyBundle(dir); err != nil {
		return diffBundle{}, err
	}
	manifestData, err := os.ReadFile(filepath.Join(dir, "manifest.json")) // #nosec G304 -- user-selected bundle.
	if err != nil {
		return diffBundle{}, fmt.Errorf("reading manifest.json: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return diffBundle{}, fmt.Errorf("decoding manifest.json: %w", err)
	}
	if manifest.Format != BundleFormat {
		return diffBundle{}, fmt.Errorf(i18n.T(
			"format de bundle non pris en charge : %q (attendu %q)",
			"unsupported bundle format: %q (expected %q)"), manifest.Format, BundleFormat)
	}

	assessmentData, err := os.ReadFile(filepath.Join(dir, "assessment.json")) // #nosec G304 -- user-selected bundle.
	if err != nil {
		return diffBundle{}, fmt.Errorf("reading assessment.json: %w", err)
	}
	var report assessment.Assessment
	if err := json.Unmarshal(assessmentData, &report); err != nil {
		return diffBundle{}, fmt.Errorf("decoding assessment.json: %w", err)
	}
	if manifest.Tool != report.Run.Tool || manifest.Ruleset != report.Run.Ruleset ||
		manifest.Target != report.Run.Target || manifest.Source != report.Run.Source {
		return diffBundle{}, errors.New(i18n.T(
			"le manifeste et l'assessment se contredisent sur la provenance",
			"manifest and assessment disagree about provenance"))
	}

	info := BundleInfo{
		Generated:       manifest.Generated,
		Tool:            manifest.Tool,
		Ruleset:         manifest.Ruleset,
		Target:          manifest.Target,
		Source:          manifest.Source,
		Format:          manifest.Format,
		InventorySchema: manifest.InventorySchema,
	}
	bundle := diffBundle{info: info, assessment: report}

	inputData, err := os.ReadFile(filepath.Join(dir, "input.json")) // #nosec G304 -- user-selected bundle.
	if errors.Is(err, os.ErrNotExist) {
		return bundle, nil
	}
	if err != nil {
		return diffBundle{}, fmt.Errorf("reading input.json: %w", err)
	}
	var input model.Posture
	if err := json.Unmarshal(inputData, &input); err != nil {
		return diffBundle{}, fmt.Errorf("decoding input.json: %w", err)
	}
	if input.Provider != "" && input.Provider != manifest.Target.Provider {
		return diffBundle{}, fmt.Errorf(i18n.T(
			"provider de input.json %q différent du manifeste %q",
			"input.json provider %q differs from manifest %q"), input.Provider, manifest.Target.Provider)
	}
	bundle.resources = input.Resources
	bundle.hasInput = true
	return bundle, nil
}

func compareBundleInfo(before, after BundleInfo) []VersionChange {
	var changes []VersionChange
	add := func(component, old, current string) {
		if old != current {
			changes = append(changes, VersionChange{Component: component, Before: old, After: current})
		}
	}
	add("tool", componentVersion(before.Tool), componentVersion(after.Tool))
	add("ruleset", componentVersion(before.Ruleset), componentVersion(after.Ruleset))
	add("bundle_format", before.Format, after.Format)
	add("inventory_schema", before.InventorySchema, after.InventorySchema)
	add("source", before.Source, after.Source)
	add("region", before.Target.Region, after.Target.Region)
	add("platform", before.Target.Platform, after.Target.Platform)
	return changes
}

func componentVersion(component assessment.Component) string {
	return fmt.Sprintf("%s@%s (%s)", component.Name, component.Version, component.Digest)
}

func compareResults(before, after []assessment.Result) ([]ResultChange, DiffSummary, error) {
	oldResults, err := indexResults(before)
	if err != nil {
		return nil, DiffSummary{}, fmt.Errorf("indexing before assessment: %w", err)
	}
	newResults, err := indexResults(after)
	if err != nil {
		return nil, DiffSummary{}, fmt.Errorf("indexing after assessment: %w", err)
	}

	keys := make(map[resultKey]struct{}, len(oldResults)+len(newResults))
	for key := range oldResults {
		keys[key] = struct{}{}
	}
	for key := range newResults {
		keys[key] = struct{}{}
	}
	ordered := make([]resultKey, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].control != ordered[j].control {
			return ordered[i].control < ordered[j].control
		}
		return ordered[i].subject < ordered[j].subject
	})

	var changes []ResultChange
	var summary DiffSummary
	for _, key := range ordered {
		oldResult, hadOld := oldResults[key]
		newResult, hasNew := newResults[key]
		if hadOld && hasNew && oldResult.Status == newResult.Status {
			continue
		}
		change := ResultChange{Control: key.control, Subject: key.subject}
		switch {
		case !hadOld:
			change.Kind = "added"
			change.After = newResult.Status
			summary.ResultsAdded++
		case !hasNew:
			change.Kind = "removed"
			change.Before = oldResult.Status
			summary.ResultsRemoved++
		default:
			change.Before = oldResult.Status
			change.After = newResult.Status
			change.Kind = resultChangeKind(oldResult.Status, newResult.Status)
			switch change.Kind {
			case "fixed":
				summary.Fixed++
			case "regressed":
				summary.Regressed++
			case "visibility-lost":
				summary.VisibilityLost++
			case "visibility-restored":
				summary.VisibilityRestored++
			default:
				summary.StatusChanged++
			}
		}
		changes = append(changes, change)
	}
	return changes, summary, nil
}

func indexResults(results []assessment.Result) (map[resultKey]assessment.Result, error) {
	index := make(map[resultKey]assessment.Result, len(results))
	for _, result := range results {
		if result.Control == "" {
			return nil, errors.New("assessment result has an empty control identifier")
		}
		key := resultKey{control: result.Control, subject: result.Subject}
		if _, exists := index[key]; exists {
			return nil, fmt.Errorf("duplicate assessment result for control %q and subject %q", key.control, key.subject)
		}
		index[key] = result
	}
	return index, nil
}

func resultChangeKind(before, after assessment.Status) string {
	switch {
	case before == assessment.Fail && after == assessment.Pass:
		return "fixed"
	case before == assessment.Pass && after == assessment.Fail:
		return "regressed"
	case isVisible(before) && !isVisible(after):
		return "visibility-lost"
	case !isVisible(before) && isVisible(after):
		return "visibility-restored"
	default:
		return "status-changed"
	}
}

func isVisible(status assessment.Status) bool {
	return status != assessment.NotEvaluated && status != assessment.Error
}

func compareResources(before, after []model.Resource) ([]ResourceChange, error) {
	oldResources, err := indexResources(before)
	if err != nil {
		return nil, fmt.Errorf("indexing before inventory: %w", err)
	}
	newResources, err := indexResources(after)
	if err != nil {
		return nil, fmt.Errorf("indexing after inventory: %w", err)
	}
	keys := make(map[resourceKey]struct{}, len(oldResources)+len(newResources))
	for key := range oldResources {
		keys[key] = struct{}{}
	}
	for key := range newResources {
		keys[key] = struct{}{}
	}
	ordered := make([]resourceKey, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].provider != ordered[j].provider {
			return ordered[i].provider < ordered[j].provider
		}
		if ordered[i].typeName != ordered[j].typeName {
			return ordered[i].typeName < ordered[j].typeName
		}
		return ordered[i].id < ordered[j].id
	})

	var changes []ResourceChange
	for _, key := range ordered {
		oldResource, hadOld := oldResources[key]
		newResource, hasNew := newResources[key]
		if hadOld == hasNew {
			continue
		}
		resource := oldResource
		kind := "removed"
		if hasNew {
			resource = newResource
			kind = "added"
		}
		changes = append(changes, ResourceChange{
			Provider: resource.Provider,
			Type:     resource.Type,
			ID:       resource.ID,
			Kind:     kind,
		})
	}
	return changes, nil
}

func indexResources(resources []model.Resource) (map[resourceKey]model.Resource, error) {
	index := make(map[resourceKey]model.Resource, len(resources))
	for _, resource := range resources {
		if resource.Type == "" || resource.ID == "" {
			return nil, errors.New("inventory resource has an empty type or ID")
		}
		key := resourceKey{provider: resource.Provider, typeName: resource.Type, id: resource.ID}
		if _, exists := index[key]; exists {
			return nil, fmt.Errorf("duplicate inventory resource %s/%s/%s", resource.Provider, resource.Type, resource.ID)
		}
		index[key] = resource
	}
	return index, nil
}
