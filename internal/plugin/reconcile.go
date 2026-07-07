package plugin

import "github.com/darkquasar/patronus/internal/lock"

// Reconcile updates plugin lock entries against what tools report installed.
// idsByTool maps tool -> set of installed LOCK-ENTRY NAMES for that tool (the
// caller has already resolved each entry's per-tool name@marketplace id and
// reduced it to the entry name). An entry becomes:
//   - verified, if any reachable tool reports it installed;
//   - missing,  if at least one tool was reachable but none report it;
//   - unverified (unchanged), if NO tool was reachable (empty idsByTool).
//
// Non-plugin entries are returned untouched.
func Reconcile(entries []lock.Entry, idsByTool map[string]map[string]bool) []lock.Entry {
	anyReachable := len(idsByTool) > 0
	out := make([]lock.Entry, len(entries))
	copy(out, entries)
	for i := range out {
		if out[i].Kind != "plugin" {
			continue
		}
		if !anyReachable {
			continue // cannot confirm; leave as-is (typically unverified)
		}
		found := false
		for _, ids := range idsByTool {
			if ids[out[i].Name] {
				found = true
				break
			}
		}
		if found {
			out[i].Status = lock.StatusVerified
		} else {
			out[i].Status = lock.StatusMissing
		}
	}
	return out
}
