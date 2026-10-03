package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/install"
	"github.com/darkquasar/patronus/internal/packagebundle"
	"github.com/darkquasar/patronus/internal/packagedelivery"
	"github.com/darkquasar/patronus/internal/packagestate"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/scan"
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
	m, err := acquireMutation(home, filepath.Join(home, ".patronus/state.json"))
	if err != nil {
		return result, err
	}
	defer m.close(&err)
	return deployDirectoriesLocked(ctx, home, cs, force, m, nil)
}

func deployDirectoriesLocked(ctx context.Context, home string, cs *diff.ChangeSet, force bool, m *mutation, prepared packagedelivery.Fetcher) (result directoryDeployResult, err error) {
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
	service := directoryServiceForDeploy(home)
	if prepared != nil {
		service.Fetcher = prepared
	}
	if service.Fetcher == nil {
		return result, errors.New("directory package fetcher is not configured")
	}
	for _, req := range requests {
		if err = recoverDirectory(ctx, service, req.Recipe, m); err != nil {
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
			if refErr := repairDirectoryReference(home, outcome.Receipt, m); refErr != nil {
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
func recoverDirectory(ctx context.Context, service *packagedelivery.Service, name string, m *mutation) error {
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
		if err := repairDirectoryReference(service.Home, receipt, m); err != nil {
			return errors.Join(recoveryErr, fmt.Errorf("package %s: repair recovered discovery reference: %w", name, err))
		}
	}
	if recoveryErr == nil {
		resolved, err := packagestate.ReadTransaction(service.Home, name)
		if err != nil {
			return err
		}
		if resolved != nil && resolved.Operation == "remove" && resolved.Phase == packagestate.Committed {
			return acknowledgeDirectoryRemoval(service.Home, name, receipt, m)
		}
	}
	return recoveryErr
}

// acknowledgeDirectoryRemoval runs only after service recovery proves the
// reduced receipt committed. Keep evidence until discovery and its link are synced.
func acknowledgeDirectoryRemoval(home, name string, receipt *packagestate.Receipt, m *mutation) error {
	path := filepath.Join(home, ".patronus", "state.json")
	if err := m.check(path); err != nil {
		return err
	}
	if receipt != nil {
		if err := repairDirectoryReference(home, receipt, m); err != nil {
			return err
		}
	} else {
		st, err := state.Load(path)
		if err != nil {
			return err
		}
		remaining := st.Items[:0]
		for _, item := range st.Items {
			if item.PackageReceipt == name {
				continue
			}
			remaining = append(remaining, item)
		}
		if len(remaining) != len(st.Items) {
			st.Items = remaining
			if err := m.saveState(path, st, nil); err != nil {
				return err
			}
		}
	}
	for _, target := range []string{path, filepath.Dir(path)} {
		f, err := os.Open(target)
		if os.IsNotExist(err) && target == path {
			continue
		}
		if err != nil {
			return err
		}
		syncErr := f.Sync()
		closeErr := f.Close()
		if err := errors.Join(syncErr, closeErr); err != nil {
			return err
		}
	}
	return packagestate.ClearTransaction(home, name)
}

// mergeDirectoryDiscovery adds receipt-only and journal-only identities in memory.
func mergeDirectoryDiscovery(home string, st *state.State) error {
	packages, err := scan.Packages(home)
	if err != nil {
		return err
	}
	for _, pkg := range packages {
		state.Merge(st, []state.Item{{Artifact: pkg.Recipe, ItemVersion: pkg.Version, PackageReceipt: pkg.Recipe, Type: "install-only", Tool: recipe.TargetAgnostic, Scope: "global"}})
	}
	return nil
}

// repairDirectoryReference is called only for committed ownership under the package lock.
func repairDirectoryReference(home string, receipt *packagestate.Receipt, m *mutation) error {
	path := filepath.Join(home, ".patronus", "state.json")
	if err := m.check(path); err != nil {
		return err
	}
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
	return m.saveState(path, s, nil)
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

// dp06Acquisition owns only disposable, verified input bytes. The existing
// appliers retain all destination/receipt/state authority and reverify replay.
type dp06Acquisition struct{ paths map[string]string }

func (a *dp06Acquisition) Fetch(ctx context.Context, address string) (io.ReadCloser, error) {
	return a.Open(ctx, address)
}
func (a *dp06Acquisition) Open(ctx context.Context, address string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, ok := a.paths[address]
	if !ok {
		return nil, fmt.Errorf("unprepared acquisition %s", address)
	}
	return os.Open(path)
}
func (a *dp06Acquisition) close() {
	for _, path := range a.paths {
		os.Remove(path)
	}
}
func dp06Acquire(ctx context.Context, cs *diff.ChangeSet) (*dp06Acquisition, error) {
	a := &dp06Acquisition{paths: map[string]string{}}
	fail := func(err error) (*dp06Acquisition, error) {
		a.close()
		return nil, fmt.Errorf("whole-selection acquisition: %w; no selected destination/state/lock written", err)
	}
	for _, d := range cs.Diffs {
		var path, address string
		if d.Directory != nil {
			req := directoryRequest(d.Directory)
			var err error
			path, err = packagedelivery.AcquireArchive(ctx, req, directoryFetcherForDeploy)
			if err != nil {
				return fail(err)
			}
			address = req.URL
		} else if d.Fetch != nil && d.Action != diff.Skip {
			data, err := install.AcquireFetch(ctx, d.Fetch, fetcherForDeploy)
			if err != nil {
				return fail(err)
			}
			f, err := os.CreateTemp("", "patronus-fetch-*")
			if err != nil {
				return fail(err)
			}
			path = f.Name()
			address = d.Fetch.URL
			_, writeErr := f.Write(data)
			closeErr := f.Close()
			if err := errors.Join(writeErr, closeErr); err != nil {
				os.Remove(path)
				return fail(err)
			}
		} else {
			continue
		}
		if old, ok := a.paths[address]; ok {
			oldBytes, readErr := os.ReadFile(old)
			if readErr != nil {
				os.Remove(path)
				return fail(readErr)
			}
			newBytes, readErr := os.ReadFile(path)
			if readErr != nil {
				os.Remove(path)
				return fail(readErr)
			}
			if sha256.Sum256(oldBytes) != sha256.Sum256(newBytes) {
				os.Remove(path)
				return fail(fmt.Errorf("inconsistent bytes for repeated acquisition URL %s", address))
			}
			os.Remove(old)
		}
		a.paths[address] = path
	}
	return a, nil
}
