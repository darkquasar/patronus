package diff

import "testing"

func TestClassifyCreate(t *testing.T) {
	cases := []struct {
		name          string
		before, after []byte
		exists        bool
		want          Action
	}{
		{"absent", nil, []byte("x"), false, Create},
		{"identical", []byte("x"), []byte("x"), true, Skip},
		{"differs", []byte("x"), []byte("y"), true, Conflict},
	}
	for _, c := range cases {
		if got := Classify(Create, c.before, c.after, c.exists); got != c.want {
			t.Errorf("%s: Classify(Create) = %s, want %s", c.name, got, c.want)
		}
	}
}

func TestClassifyAppendMerge(t *testing.T) {
	// Non-destructive: After folds in Before, so equal -> SKIP, differ -> keep.
	for _, intended := range []Action{Append, Merge} {
		if got := Classify(intended, []byte("a"), []byte("a"), true); got != Skip {
			t.Errorf("Classify(%s) equal = %s, want SKIP", intended, got)
		}
		if got := Classify(intended, []byte("a"), []byte("a\nb"), true); got != intended {
			t.Errorf("Classify(%s) differ = %s, want %s", intended, got, intended)
		}
		// Never CONFLICT even when the target exists and differs.
		if got := Classify(intended, []byte("a"), []byte("z"), true); got == Conflict {
			t.Errorf("Classify(%s) must never CONFLICT", intended)
		}
	}
}

func TestUnified(t *testing.T) {
	// Content change -> hunk markers.
	d := FileDiff{Action: Append, Before: []byte("a\nb\n"), After: []byte("a\nc\n")}
	u := d.Unified()
	if !contains(u, "@@") || !contains(u, "-b") || !contains(u, "+c") {
		t.Errorf("unexpected unified diff:\n%s", u)
	}
	// CREATE from nothing -> all added.
	create := FileDiff{Action: Create, After: []byte("x\ny\n")}
	if cu := create.Unified(); !contains(cu, "+x") || !contains(cu, "+y") {
		t.Errorf("create unified diff:\n%s", cu)
	}
	// SKIP and IsDir -> empty.
	if (FileDiff{Action: Skip, Before: []byte("a"), After: []byte("a")}).Unified() != "" {
		t.Error("skip should be empty")
	}
	if (FileDiff{Action: Create, IsDir: true, After: []byte("x")}).Unified() != "" {
		t.Error("dir row should be empty")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestCounts(t *testing.T) {
	cs := &ChangeSet{Diffs: []FileDiff{
		{Action: Create},
		{Action: Create},
		{Action: Skip},
		{Action: Create, IsDir: true}, // excluded
	}}
	c := cs.Counts()
	if c[Create] != 2 || c[Skip] != 1 {
		t.Errorf("counts = %v", c)
	}
}

// TestCountsCountsEveryContributor pins the tally to what --deploy will actually
// APPLY, not to how many FileDiffs happen to represent it.
//
// A composed file folds many artifacts into ONE FileDiff: six instructions APPEND
// into one CLAUDE.md, and eight artifacts MERGE into one settings.json (7 hooks + a
// statusline + native-sandbox, which turns Claude's sandbox on). The plan TABLE
// renders a row per contributor — so a footer that counted diffs said "1 APPEND,
// 2 MERGE" directly beneath eight visible MERGE rows, contradicting itself and
// under-reporting the change the user is being asked to approve.
func TestCountsCountsEveryContributor(t *testing.T) {
	cs := &ChangeSet{Diffs: []FileDiff{
		{
			Action: Append, Path: "/h/CLAUDE.md", Artifact: "agents-spine",
			Contrib: []SectionContrib{
				{Artifact: "agent-rules"},
				{Artifact: "go-style-uber"},
			},
		},
		{
			Action: Merge, Path: "/h/.claude/settings.json", Artifact: "skills-dispatch-activate",
			SettingContrib: []SettingContrib{
				{Artifact: "block-secrets"},
				{Artifact: "native-sandbox"},
			},
		},
		{Action: Create, Path: "/h/.claude/skills/tdd/SKILL.md", Artifact: "tdd"},
	}}

	c := cs.Counts()
	if got, want := c[Append], 3; got != want {
		t.Errorf("APPEND count = %d, want %d (the owner + 2 folded-in contributors)", got, want)
	}
	if got, want := c[Merge], 3; got != want {
		t.Errorf("MERGE count = %d, want %d (the owner + 2 folded-in contributors)", got, want)
	}
	if got, want := c[Create], 1; got != want {
		t.Errorf("CREATE count = %d, want %d", got, want)
	}
}
