package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/packagebundle"
	"github.com/darkquasar/patronus/internal/packagedelivery"
	"github.com/darkquasar/patronus/internal/packagestate"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/state"
)

var directoryFetcherForDeploy packagedelivery.Fetcher = packagedelivery.HTTPSFetcher{}
var directoryLookPath = exec.LookPath

// directoryServiceForDeploy permits transaction faults without live network or processes.
var directoryServiceForDeploy = func(home string) *packagedelivery.Service {
	return &packagedelivery.Service{Home: home, Fetcher: directoryFetcherForDeploy}
}

type directoryDeployResult struct {
	Legacy           *diff.ChangeSet
	Written, Skipped int
}

func directoryRequest(d *diff.DirectorySpec) packagedelivery.Request {
	return packagedelivery.Request{Recipe: d.Recipe, RecipeVersion: d.RecipeVersion, URL: d.URL, SHA256: "sha256:" + strings.ToLower(strings.TrimPrefix(d.SHA256, "sha256:")), Root: d.Root, Identity: packagebundle.Identity{Name: d.PackageName, Version: d.PackageVersion, OS: d.OS, Arch: d.Arch}}
}

func directoryRequests(cs *diff.ChangeSet) []packagedelivery.Request {
	var requests []packagedelivery.Request
	seen := map[string]bool{}
	for _, d := range cs.Diffs {
		if d.Directory == nil || seen[d.Directory.Recipe] {
			continue
		}
		seen[d.Directory.Recipe] = true
		requests = append(requests, directoryRequest(d.Directory))
	}
	return requests
}

func directoryConflict(req packagedelivery.Request, in packagedelivery.Inspection, force bool) error {
	if len(in.Unknown) > 0 {
		return &packagedelivery.ConflictError{Recipe: req.Recipe, Kind: "unknown-content", Paths: in.Unknown}
	}
	if len(in.Changed) > 0 && !force {
		return &packagedelivery.ConflictError{Recipe: req.Recipe, Kind: "owned-drift", Paths: in.Changed}
	}
	return nil
}

func directoryDiagnostic(err error) error {
	var conflict *packagedelivery.ConflictError
	if errors.As(err, &conflict) && conflict.Kind == "owned-drift" {
		return fmt.Errorf("%w; to replace owned edits, retry: patronus install %s --deploy --force (or patronus update %s --deploy --force)", err, conflict.Recipe, conflict.Recipe)
	}
	return err
}

// inspectDirectoryPlan annotates intent without downloading, locking or recovering.
func inspectDirectoryPlan(home string, cs *diff.ChangeSet) error {
	service := &packagedelivery.Service{Home: home}
	for i := range cs.Diffs {
		d := &cs.Diffs[i]
		if d.Directory == nil {
			continue
		}
		req := directoryRequest(d.Directory)
		in, err := service.Inspect(req)
		if err != nil {
			var conflict *packagedelivery.ConflictError
			if !errors.As(err, &conflict) {
				return err
			}
			d.Note = directoryDiagnostic(err).Error()
			continue
		}
		switch {
		case in.Pending != nil:
			d.Note = "pending recovery: " + string(in.Pending.Phase) + "; deploy retries recovery under the package lock"
		case directoryConflict(req, in, false) != nil:
			d.Note = directoryDiagnostic(directoryConflict(req, in, false)).Error()
		case len(in.Missing) > 0:
			d.Note = "missing owned paths: " + strings.Join(in.Missing, ", ")
		case in.Receipt != nil && in.Receipt.RecipeVersion == req.RecipeVersion && in.Receipt.Identity == req.Identity && in.Receipt.ArchiveSHA256 == req.SHA256 && in.Receipt.URL == req.URL:
			d.Note = "unchanged; verify owned package"
		default:
			d.Note = "fetch verified static package"
		}
	}
	return nil
}

// preflightDirectories checks the entire batch before any selected mutation.
// Pending selected work is inspected again after locked recovery.
func preflightDirectories(home string, cs *diff.ChangeSet, force bool) error {
	service := &packagedelivery.Service{Home: home}
	for _, req := range directoryRequests(cs) {
		in, err := service.Inspect(req)
		if err != nil {
			return directoryDiagnostic(err)
		}
		if err := directoryConflict(req, in, force); err != nil {
			return directoryDiagnostic(err)
		}
	}
	return nil
}

func deployDirectories(ctx context.Context, home string, cs *diff.ChangeSet, force bool) (result directoryDeployResult, err error) {
	result.Legacy = &diff.ChangeSet{DryRun: cs.DryRun}
	for _, d := range cs.Diffs {
		if d.Directory == nil {
			result.Legacy.Diffs = append(result.Legacy.Diffs, d)
		}
	}
	requests := directoryRequests(cs)
	if len(requests) == 0 {
		result.Legacy = cs
		return result, nil
	}
	if err = preflightDirectories(home, cs, force); err != nil {
		return result, err
	}
	release, err := packagestate.Acquire(home)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, release()) }()
	service := directoryServiceForDeploy(home)
	if service.Fetcher == nil {
		return result, errors.New("directory package fetcher is not configured")
	}
	for _, req := range requests {
		if err = recoverDirectory(ctx, service, req.Recipe); err != nil {
			return result, directoryDiagnostic(err)
		}
	}
	if err = preflightDirectories(home, cs, force); err != nil {
		return result, err
	}
	for _, req := range requests {
		outcome, replaceErr := service.Replace(ctx, req, force)
		// A committed cleanup error still carries the authoritative receipt.
		if outcome.Receipt != nil {
			if outcome.Mutated {
				result.Written++
			} else {
				result.Skipped++
			}
			if refErr := repairDirectoryReference(home, outcome.Receipt); refErr != nil {
				return result, errors.Join(replaceErr, fmt.Errorf("package %s committed; repair discovery reference by retrying install: %w", req.Recipe, refErr))
			}
		}
		if replaceErr != nil {
			return result, directoryDiagnostic(replaceErr)
		}
	}
	return result, nil
}

// recoverDirectory repairs discovery from the resolved committed receipt before
// the next request can fail. A known committed cleanup failure also permits repair.
func recoverDirectory(ctx context.Context, service *packagedelivery.Service, name string) error {
	tx, err := packagestate.ReadTransaction(service.Home, name)
	if err != nil {
		return err
	}
	recoveryErr := service.Recover(ctx, name)
	receipt, err := packagestate.Load(service.Home, name)
	if err != nil {
		return errors.Join(recoveryErr, err)
	}
	committed := tx != nil && (tx.Phase == packagestate.Committed || (tx.Phase == packagestate.RecoveryRequired && tx.ResumePhase == packagestate.Committed)) && tx.Operation != "remove" && reflect.DeepEqual(receipt, tx.Candidate)
	if receipt != nil && (recoveryErr == nil || committed) {
		if err := repairDirectoryReference(service.Home, receipt); err != nil {
			return errors.Join(recoveryErr, fmt.Errorf("package %s: repair recovered discovery reference: %w", name, err))
		}
	}
	return recoveryErr
}

// repairDirectoryReference is called only for committed ownership under the package lock.
func repairDirectoryReference(home string, receipt *packagestate.Receipt) error {
	path := filepath.Join(home, ".patronus", "state.json")
	s, err := state.Load(path)
	if err != nil {
		return err
	}
	item := directoryStateItem(receipt)
	for _, existing := range s.Items {
		if existing.Artifact == item.Artifact && existing.Tool == item.Tool && existing.Scope == item.Scope {
			item.InstalledAt = existing.InstalledAt
			if reflect.DeepEqual(existing, item) {
				return nil
			}
			break
		}
	}
	state.Merge(s, []state.Item{item})
	return state.Save(path, s)
}

func directoryStateItem(receipt *packagestate.Receipt) state.Item {
	return state.Item{Artifact: receipt.Recipe, ItemVersion: receipt.RecipeVersion, PackageReceipt: receipt.Recipe, Type: "install-only", Tool: recipe.TargetAgnostic, Scope: "global", InstalledAt: time.Now().UTC().Format(time.RFC3339)}
}

func printDirectoryReadiness(out io.Writer, cs *diff.ChangeSet) {
	for _, req := range directoryRequests(cs) {
		if req.Recipe != "pi-sandbox" {
			continue
		}
		fmt.Fprintln(out, "Running this kit requires sbx and separate provider authentication.")
		fmt.Fprintf(out, "Package root: %s\nRead launch instructions: %s\n", req.Root, filepath.Join(req.Root, "README.md"))
		if _, err := directoryLookPath("sbx"); err != nil {
			fmt.Fprintln(out, "Package installed; install sbx before using it.")
		}
	}
}
