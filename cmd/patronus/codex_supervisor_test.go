//go:build artifactbehavior

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// These tests execute the bundled plan-execute-parallel-cx supervisor against
// invented fake-worker processes in scratch Git repositories. They never call a
// Codex, Claude or Pi backend, and they do not qualify native Codex workers.

var cxSvSkillDir = filepath.Join("..", "..", "artifacts", "skills", "plan-execute-parallel-cx")

type cxSvAttempt struct {
	Status              string                     `json:"status"`
	Accepted            bool                       `json:"accepted"`
	Worktree            string                     `json:"worktree"`
	Pgid                int                        `json:"pgid"`
	Patch               *string                    `json:"patch"`
	EvidencePatch       string                     `json:"evidencePatch"`
	EvidencePatchSha256 string                     `json:"evidencePatchSha256"`
	Details             map[string]json.RawMessage `json:"details"`
	Stop                struct {
		Signals   []string `json:"signals"`
		Confirmed bool     `json:"confirmed"`
	} `json:"stop"`
}

type cxSvRecord struct {
	Status            string   `json:"status"`
	RetainedWorktrees []string `json:"retainedWorktrees"`
	Tasks             map[string]struct {
		State    string        `json:"state"`
		Attempts []cxSvAttempt `json:"attempts"`
	} `json:"tasks"`
}

type cxSvScratch struct {
	t                 *testing.T
	tmp, repo, base   string
	env               []string
	python, supervise string
}

func cxSvNew(t *testing.T) *cxSvScratch {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatalf("supervisor behavior needs python3 on PATH: %v", err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Fatalf("supervisor behavior needs git on PATH: %v", err)
	}
	supervise, err := filepath.Abs(filepath.Join(cxSvSkillDir, "scripts", "supervisor.py"))
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	gitconfig := filepath.Join(tmp, "gitconfig")
	codexWrite(t, gitconfig, []byte("[user]\n\tname = Invented\n\temail = invented@example.invalid\n"))
	s := &cxSvScratch{t: t, tmp: tmp, repo: filepath.Join(tmp, "repo"), python: python, supervise: supervise,
		env: append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+gitconfig, "PYTHONDONTWRITEBYTECODE=1")}
	codexWrite(t, filepath.Join(s.repo, "README.md"), []byte("invented fixture\n"))
	s.git(s.repo, "init", "-q")
	s.git(s.repo, "add", "-A")
	s.git(s.repo, "commit", "-q", "-m", "base")
	s.base = strings.TrimSpace(s.git(s.repo, "rev-parse", "HEAD"))
	return s
}

func (s *cxSvScratch) git(dir string, args ...string) string {
	s.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir, cmd.Env = dir, s.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// run writes a one-task request and runs the supervisor, returning its exit code and run record.
func (s *cxSvScratch) run(owned []string, timeout, grace float64, worker ...string) (int, cxSvRecord) {
	s.t.Helper()
	brief := filepath.Join(s.tmp, "brief.md")
	codexWrite(s.t, brief, []byte("Invented brief.\n"))
	sum := sha256.Sum256([]byte("Invented brief.\n"))
	fake, err := filepath.Abs(filepath.Join(cxSvSkillDir, "fixtures", "fake_worker.py"))
	if err != nil {
		s.t.Fatal(err)
	}
	runDir := filepath.Join(s.tmp, "evidence")
	req := map[string]any{
		"schemaVersion": 1, "runId": "go-run", "repository": s.repo, "baseCommit": s.base,
		"brief":        map[string]string{"path": brief, "sha256": hex.EncodeToString(sum[:])},
		"worktreeRoot": filepath.Join(s.tmp, "worktrees"), "runDir": runDir, "maxConcurrency": 1,
		"tasks": []map[string]any{{"key": "w", "command": append([]string{s.python, fake}, worker...),
			"ownedPaths": owned, "timeoutSeconds": timeout, "graceSeconds": grace}},
	}
	raw, err := json.Marshal(req)
	if err != nil {
		s.t.Fatal(err)
	}
	reqPath := filepath.Join(s.tmp, "request.json")
	codexWrite(s.t, reqPath, raw)
	cmd := exec.Command(s.python, s.supervise, "run", reqPath)
	cmd.Env = s.env
	out, err := cmd.CombinedOutput()
	code := 0
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		code = exit.ExitCode()
	} else if err != nil {
		s.t.Fatalf("supervisor did not run: %v\n%s", err, out)
	}
	s.t.Logf("supervisor exit=%d output=%s", code, out)
	body, err := os.ReadFile(filepath.Join(runDir, "run.json"))
	if err != nil {
		s.t.Fatalf("no run record: %v\n%s", err, out)
	}
	var rec cxSvRecord
	if err := json.Unmarshal(body, &rec); err != nil {
		s.t.Fatal(err)
	}
	return code, rec
}

func (s *cxSvScratch) assertLeadUntouched() {
	s.t.Helper()
	if st := s.git(s.repo, "status", "--porcelain", "--untracked-files=all", "--ignored"); st != "" {
		s.t.Fatalf("lead checkout changed:\n%s", st)
	}
	if head := strings.TrimSpace(s.git(s.repo, "rev-parse", "HEAD")); head != s.base {
		s.t.Fatalf("lead HEAD drifted to %s", head)
	}
}

func cxSvAssertRetained(t *testing.T, att cxSvAttempt, rel, want string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(att.Worktree, rel))
	if err != nil || string(got) != want {
		t.Fatalf("retained worktree %s lacks %s=%q: %q %v", att.Worktree, rel, want, got, err)
	}
	patch, err := os.ReadFile(att.EvidencePatch)
	if err != nil {
		t.Fatal(err)
	}
	if sum := sha256.Sum256(patch); hex.EncodeToString(sum[:]) != att.EvidencePatchSha256 {
		t.Fatal("evidence patch hash does not match its record")
	}
}

// cxSvGroupMembers lists live non-zombie members of a process group via /proc.
// Elsewhere it reports nil, and the supervisor's confirmed-stop record stands alone.
func cxSvGroupMembers(t *testing.T, pgid int) []int {
	t.Helper()
	entries, err := os.ReadDir("/proc")
	if err != nil {
		t.Logf("no /proc; relying on the supervisor's confirmed stop for group %d", pgid)
		return nil
	}
	var members []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		raw, err := os.ReadFile(filepath.Join("/proc", e.Name(), "stat"))
		if err != nil {
			continue
		}
		fields := strings.Fields(string(raw[strings.LastIndexByte(string(raw), ')')+1:]))
		if len(fields) > 2 && fields[0] != "Z" && fields[2] == strconv.Itoa(pgid) {
			members = append(members, pid)
		}
	}
	return members
}

func TestCodexParallelSupervisorTimeoutRetainsWork(t *testing.T) {
	s := cxSvNew(t)
	code, rec := s.run([]string{"src/slow.txt"}, 1, 0.5,
		"--write", "src/slow.txt=partial\n", "--ignore-sigterm", "--sleep", "60")
	if code != 1 || rec.Status != "failed" || rec.Tasks["w"].State != "failed" {
		t.Fatalf("timeout must fail the task: exit=%d %+v", code, rec)
	}
	att := rec.Tasks["w"].Attempts[0]
	if att.Status != "timeout" || att.Accepted || att.Patch != nil {
		t.Fatalf("timed-out attempt accepted or captured: %+v", att)
	}
	if !att.Stop.Confirmed || strings.Join(att.Stop.Signals, ",") != "SIGTERM,SIGKILL" {
		t.Fatalf("stop not escalated and confirmed: %+v", att.Stop)
	}
	if live := cxSvGroupMembers(t, att.Pgid); len(live) != 0 {
		t.Fatalf("worker group still live after confirmed stop: %v", live)
	}
	cxSvAssertRetained(t, att, "src/slow.txt", "partial\n")
	if len(rec.RetainedWorktrees) != 1 || rec.RetainedWorktrees[0] != att.Worktree {
		t.Fatalf("failed attempt not recorded as retained: %v", rec.RetainedWorktrees)
	}
	s.assertLeadUntouched()
}

func TestCodexParallelSupervisorRejectsOwnershipViolation(t *testing.T) {
	s := cxSvNew(t)
	code, rec := s.run([]string{"src/own.txt"}, 20, 0.5,
		"--write", "src/own.txt=mine\n", "--write", "README.md=stolen\n")
	if code != 1 || rec.Tasks["w"].State != "failed" {
		t.Fatalf("ownership violation must fail the task: exit=%d %+v", code, rec)
	}
	att := rec.Tasks["w"].Attempts[0]
	if att.Status != "ownership-violation" || att.Accepted || att.Patch != nil {
		t.Fatalf("non-owned write accepted: %+v", att)
	}
	var nonOwned []string
	if err := json.Unmarshal(att.Details["nonOwned"], &nonOwned); err != nil || len(nonOwned) != 1 || nonOwned[0] != "README.md" {
		t.Fatalf("violation evidence = %s (%v)", att.Details["nonOwned"], err)
	}
	cxSvAssertRetained(t, att, "README.md", "stolen\n")
	s.assertLeadUntouched()
}
