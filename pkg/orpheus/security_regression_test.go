// security_regression_test.go: regression tests for defects found in the
// path, string and permission validation layer.
//
// Copyright (c) 2026 AGILira - A. Giordano
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package orpheus

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// A value validated as a string must not be served from the cache as a
// validated path.
func TestSecurityRegression_CacheSeparatesStringAndPathValidation(t *testing.T) {
	v := NewInputValidator(DefaultValidationConfig())
	_ = v.ValidateStringFlag("user-input", "/etc/shadow")
	if r := v.ValidatePathFlag("user-input", "/etc/shadow"); r.IsValid {
		t.Errorf("ValidatePathFlag accepted /etc/shadow after ValidateStringFlag cached it: %s", r)
	}

	v = NewInputValidator(DefaultValidationConfig())
	_ = v.ValidateStringFlag("file", "/etc/shadow")
	if r := v.ValidateFileFlag("/etc/shadow"); r.IsValid {
		t.Errorf("ValidateFileFlag accepted /etc/shadow via cache: %s", r)
	}
}

// The cache must still serve repeated validations of the same kind.
func TestSecurityRegression_CacheStillHits(t *testing.T) {
	v := NewInputValidator(DefaultValidationConfig())
	_ = v.ValidatePathFlag("f", "data/x.json")
	_ = v.ValidateStringFlag("f", "data/x.json")
	if n := v.GetCacheStats()["cache_size"]; n != 2 {
		t.Fatalf("cache_size = %d, want 2 (one entry per kind)", n)
	}
	_ = v.ValidatePathFlag("f", "data/x.json")
	if n := v.GetCacheStats()["cache_size"]; n != 2 {
		t.Errorf("repeat lookup added an entry: cache_size = %d", n)
	}
}

// The default denied entries /home/*/.ssh and /home/*/.gnupg must match a
// real home directory.
func TestSecurityRegression_DeniedGlobEntriesMatch(t *testing.T) {
	for _, p := range []string{"/home/alice/.ssh/id_rsa", "/home/alice/.ssh", "/home/bob/.gnupg/pubring.kbx"} {
		if r := ValidateSecurePath(p, DefaultSecurityConfig()); r.IsValid {
			t.Errorf("%s accepted although denied by a /home/*/ entry", p)
		}
	}
	if r := ValidateSecurePath("/home/alice/.sshconfig", DefaultSecurityConfig()); !r.IsValid {
		t.Errorf("/home/alice/.sshconfig rejected: %v", r.Errors)
	}
}

// AllowedPaths must match whole path segments.
func TestSecurityRegression_AllowedPathsMatchOnBoundary(t *testing.T) {
	cfg := DefaultSecurityConfig()
	cfg.AllowedPaths = []string{"/srv/app"}
	if r := ValidateSecurePath("/srv/app-secrets/key.pem", cfg); r.IsValid {
		t.Errorf("/srv/app-secrets/key.pem accepted with AllowedPaths=[/srv/app]")
	}
	for _, p := range []string{"/srv/app", "/srv/app/conf.yml"} {
		if r := ValidateSecurePath(p, cfg); !r.IsValid {
			t.Errorf("%s rejected with AllowedPaths=[/srv/app]: %v", p, r.Errors)
		}
	}
}

// DeniedPaths must match whole path segments: siblings that share a prefix
// are legitimate, the denied directory and its contents are not.
func TestSecurityRegression_DeniedPathsMatchOnBoundary(t *testing.T) {
	for _, p := range []string{"/development/proj/config.yml", "/binaries/tool", "/rootfs/img", "/system-data/x"} {
		if r := ValidateSecurePath(p, DefaultSecurityConfig()); !r.IsValid {
			t.Errorf("legit path %q rejected: %v", p, r.Errors)
		}
	}
	for _, p := range []string{"/etc", "/dev/sda", "/var/log/syslog"} {
		if r := ValidateSecurePath(p, DefaultSecurityConfig()); r.IsValid {
			t.Errorf("denied path %q accepted", p)
		}
	}
}

// Path flags reject shell substitution and chaining, not ordinary file names.
func TestSecurityRegression_PathFlagAcceptsOrdinaryNames(t *testing.T) {
	v := NewInputValidator(DefaultValidationConfig())
	for _, p := range []string{"/home/u/Dropbox/notes.txt", "/tmp/deleted_items.txt", "/tmp/report--v2.pdf", "/tmp/R&D/plan.txt"} {
		if r := v.ValidatePathFlag("file", p); !r.IsValid {
			t.Errorf("legit path %q rejected: %v", p, r.ValidationErrors)
		}
	}
	for _, p := range []string{"/tmp/$(id)", "/tmp/`id`", "/tmp/a;rm", "/tmp/a|nc", "/tmp/a&&b"} {
		if r := v.ValidatePathFlag("file", p); r.IsValid {
			t.Errorf("shell metacharacter path %q accepted", p)
		}
	}
}

// String flags keep the documented SQL and comment patterns.
func TestSecurityRegression_StringFlagKeepsSQLPatterns(t *testing.T) {
	v := NewInputValidator(DefaultValidationConfig())
	for _, s := range []string{"'; DROP TABLE users; --", "/* x */", "a & b"} {
		if r := v.ValidateStringFlag("q", s); r.IsValid {
			t.Errorf("string %q accepted", s)
		}
	}
}

// forbiddenSystemDir returns an OS system directory denied by
// DefaultSecurityConfig on the current platform.
func forbiddenSystemDir() string {
	if runtime.GOOS == "windows" {
		return `C:\Windows`
	}
	return "/etc"
}

// A symlink into a system directory must be judged by its target.
func TestSecurityRegression_SymlinkIntoEtcRejected(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "cfg")
	sysDir := forbiddenSystemDir()
	if err := os.Symlink(sysDir, link); err != nil {
		t.Skip(err)
	}
	v := NewInputValidator(DefaultValidationConfig())
	for _, p := range []string{filepath.Join(link, "passwd"), filepath.Join(link, "not-yet-created.conf")} {
		if r := v.ValidatePathFlag("file", p); r.IsValid {
			t.Errorf("%s (under a link to %s) accepted: %s", p, sysDir, r)
		}
	}
}

// A symlink inside an allowed directory must not lead outside it, and an
// allowed directory reached through a link must still work.
func TestSecurityRegression_SymlinkAndAllowedPaths(t *testing.T) {
	root := t.TempDir()
	allowed := filepath.Join(root, "allowed")
	outside := filepath.Join(root, "outside")
	for _, d := range []string{allowed, outside} {
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(allowed, "escape")); err != nil {
		t.Skip(err)
	}
	if err := os.Symlink(allowed, filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultSecurityConfig()
	cfg.AllowedPaths = []string{filepath.Join(root, "alias")}
	if r := ValidateSecurePath(filepath.Join(root, "alias", "escape", "f"), cfg); r.IsValid {
		t.Errorf("link out of the allowed directory accepted")
	}
	if r := ValidateSecurePath(filepath.Join(root, "alias", "f"), cfg); !r.IsValid {
		t.Errorf("file in an allowed directory reached through a link rejected: %v", r.Errors)
	}
}

// Mixed percent-encodings of ".." are traversal too.
func TestSecurityRegression_EncodedTraversalVariants(t *testing.T) {
	for _, p := range []string{"%2e./secret", ".%2E/secret", "%252e%252e/x", "%zz%2e./x", "%2e%2E/x"} {
		if r := ValidateSecurePath(p, DefaultSecurityConfig()); r.IsValid {
			t.Errorf("encoded traversal %q accepted", p)
		}
	}
	if r := ValidateSecurePath("reports/100%25.txt", DefaultSecurityConfig()); !r.IsValid {
		t.Errorf("percent sign in a name rejected: %v", r.Errors)
	}
}

// Write permission is the current user's, not the owner's.
func TestSecurityRegression_WritableIsCurrentUsers(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("needs a non-root Unix user")
	}
	const p = "/usr/share/common-licenses/GPL-3"
	if _, err := os.Stat(p); err != nil {
		t.Skip(err)
	}
	if f, err := os.OpenFile(p, os.O_WRONLY, 0); err == nil {
		_ = f.Close()
		t.Skip("file actually writable")
	}
	v := NewInputValidator(DefaultValidationConfig())
	if r := v.ValidateFileOperation(p, "write"); r.IsValid {
		t.Errorf("write approved on %s that the current user cannot write", p)
	}
	if r := v.ValidateFileOperation(p, "read"); !r.IsValid {
		t.Errorf("read refused on world-readable %s: %v", p, r.ValidationErrors)
	}
}

// A symlink is described by its target, not by its own 0777 mode.
func TestSecurityRegression_SymlinkPermissionsAreTargets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skip(err)
	}
	perm, err := AnalyzeFilePermissions(link, DefaultSecurityConfig())
	if err != nil {
		t.Fatal(err)
	}
	if perm.SecurityRisk == "critical" || perm.IsExecutable {
		t.Errorf("link to a 0600 file reported risk=%s executable=%v", perm.SecurityRisk, perm.IsExecutable)
	}
}

// ':' is an ordinary filename character outside Windows.
func TestSecurityRegression_ColonAllowedOutsideWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("NTFS alternate data streams are rejected on Windows")
	}
	p := "/tmp/backup-2026-11-24T10:30.tar"
	if r := ValidateSecurePath(p, DefaultSecurityConfig()); !r.IsValid {
		t.Errorf("legit path %q rejected: %v", p, r.Errors)
	}
}

// A dangling link is judged by where it points: writing through it creates
// the target.
func TestSecurityRegression_DanglingSymlinkJudgedByTarget(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	sysTarget := filepath.Join(forbiddenSystemDir(), "orpheus-regression-never-created")
	if err := os.Symlink(sysTarget, out); err != nil {
		t.Skip(err)
	}
	if r := ValidateSecurePath(out, DefaultSecurityConfig()); r.IsValid {
		t.Errorf("dangling link into %s accepted", sysTarget)
	}

	safe := filepath.Join(dir, "safe")
	if err := os.Symlink("missing-target", safe); err != nil {
		t.Fatal(err)
	}
	if r := ValidateSecurePath(safe, DefaultSecurityConfig()); !r.IsValid {
		t.Errorf("dangling link inside the temp dir rejected: %v", r.Errors)
	}
}

// A self-referencing link cannot be resolved; the hop limit ends the chain
// and the path is judged by its own spelling.
func TestSecurityRegression_SymlinkLoopTerminates(t *testing.T) {
	dir := t.TempDir()
	loop := filepath.Join(dir, "loop")
	if err := os.Symlink("loop", loop); err != nil {
		t.Skip(err)
	}
	if r := ValidateSecurePath(filepath.Join(loop, "f"), DefaultSecurityConfig()); !r.IsValid {
		t.Errorf("path under a self-referencing link rejected: %v", r.Errors)
	}
}

// A link whose target is itself a listed entry matches that entry directly.
func TestSecurityRegression_SymlinkTargetMatchesEntryDirectly(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "etc")
	sysDir := forbiddenSystemDir()
	if err := os.Symlink(sysDir, link); err != nil {
		t.Skip(err)
	}
	if r := ValidateSecurePath(link, DefaultSecurityConfig()); r.IsValid {
		t.Errorf("link to %s accepted", sysDir)
	}

	real := filepath.Join(dir, "real")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(dir, "alias")); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultSecurityConfig()
	cfg.AllowedPaths = []string{dir}
	if r := ValidateSecurePath(filepath.Join(dir, "alias", "f"), cfg); !r.IsValid {
		t.Errorf("link to an allowed directory rejected: %v", r.Errors)
	}
}

// A path shorter than a glob entry does not match it.
func TestSecurityRegression_GlobEntryLongerThanPath(t *testing.T) {
	if r := ValidateSecurePath("/home/alice", DefaultSecurityConfig()); !r.IsValid {
		t.Errorf("/home/alice rejected: %v", r.Errors)
	}
}

// With ValidatePathArgs off, permissions come from AnalyzeFilePermissions,
// whose own path validation can refuse the path: that becomes a warning.
func TestSecurityRegression_PermissionAnalysisWithoutPathValidation(t *testing.T) {
	cfg := DefaultValidationConfig()
	cfg.ValidatePathArgs = false
	cfg.EnableCaching = false
	v := NewInputValidator(cfg)

	r := v.ValidatePathFlag("file", "/etc/hostname")
	if _, err := os.Stat("/etc/hostname"); err != nil {
		t.Skip(err)
	}
	if !containsSubstring(r.SecurityWarnings, "Could not analyze file permissions") {
		t.Errorf("no analysis warning for a denied path: %v", r.SecurityWarnings)
	}

	path := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if r := v.ValidatePathFlag("file", path); r.FilePermissions == nil {
		t.Errorf("no permissions for %s: %v", path, r.SecurityWarnings)
	}
}

// A group-writable file carries the medium-risk warning.
func TestSecurityRegression_GroupWritableWarns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits")
	}
	path := filepath.Join(t.TempDir(), "shared.txt")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o660); err != nil {
		t.Fatal(err)
	}
	r := NewInputValidator(DefaultValidationConfig()).ValidatePathFlag("file", path)
	if !containsSubstring(r.SecurityWarnings, "medium security risk") {
		t.Errorf("no medium-risk warning for a 0660 file: %v", r.SecurityWarnings)
	}
}

func containsSubstring(list []string, sub string) bool {
	for _, s := range list {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// A relative link with a relative target must be judged by its absolute
// target, not by the relative spelling EvalSymlinks returns.
func TestSecurityRegression_RelativeSymlinkToEtcRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix /etc layout")
	}
	dir := t.TempDir()
	up := strings.Repeat("../", strings.Count(dir, string(filepath.Separator))+1)
	if err := os.Symlink(up+"etc/hostname", filepath.Join(dir, "lnk")); err != nil {
		t.Skip(err)
	}
	if err := os.Symlink(up+"etc", filepath.Join(dir, "cfg")); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	for _, p := range []string{"lnk", "cfg/passwd", "cfg/not-yet-created"} {
		if r := ValidateSecurePath(p, DefaultSecurityConfig()); r.IsValid {
			t.Errorf("relative link %q into /etc accepted", p)
		}
	}
}

// A dangling relative link is resolved against its directory's real
// location, as the kernel does, not lexically.
func TestSecurityRegression_DanglingRelativeLinkInLinkedDir(t *testing.T) {
	work := t.TempDir()
	allowed := filepath.Join(work, "allowed")
	sub := filepath.Join(work, "outside", "sub")
	for _, d := range []string{filepath.Join(allowed, "nested"), sub} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(sub, filepath.Join(allowed, "nested", "jump")); err != nil {
		t.Skip(err)
	}
	if err := os.Symlink("../evil", filepath.Join(sub, "d")); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultSecurityConfig()
	cfg.AllowedPaths = []string{allowed}
	if r := ValidateSecurePath(filepath.Join(allowed, "nested", "jump", "d"), cfg); r.IsValid {
		t.Errorf("dangling link that creates %s accepted", filepath.Join(work, "outside", "evil"))
	}
}

// With caching off, no cache key is built: memory per call must not grow
// with the flag name.
func TestSecurityRegression_NoCacheKeyWhenCachingOff(t *testing.T) {
	cfg := DefaultValidationConfig()
	cfg.EnableCaching = false
	v := NewInputValidator(cfg)
	short := "n"
	long := strings.Repeat("n", 1<<16)
	bytesPerOp := func(name string) int64 {
		const calls = 100
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		for range calls {
			_ = v.ValidateStringFlag(name, "value")
			_ = v.ValidatePathFlag(name, "data/x.json")
		}
		runtime.ReadMemStats(&after)
		return int64(after.TotalAlloc-before.TotalAlloc) / calls
	}
	// A key per call would add 2 x 64 KiB; the margin absorbs other goroutines.
	if grow := bytesPerOp(long) - bytesPerOp(short); grow > 16<<10 {
		t.Errorf("caching off still allocates %d bytes per call for the cache key", grow)
	}
}

// A relative path is judged by where it points from the working directory.
func TestSecurityRegression_RelativePathFromEtcRejected(t *testing.T) {
	if _, err := os.Stat("/etc/passwd"); err != nil {
		t.Skip(err)
	}
	t.Chdir("/etc")
	if r := ValidateSecurePath("passwd", DefaultSecurityConfig()); r.IsValid {
		t.Errorf(`"passwd" from /etc accepted`)
	}
}

func TestSecurityRegression_RelativePathFromRootRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix layout")
	}
	t.Chdir("/")
	for _, p := range []string{"etc/hostname", "etc"} {
		if r := ValidateSecurePath(p, DefaultSecurityConfig()); r.IsValid {
			t.Errorf("%q from / accepted", p)
		}
	}
}

// Relative allowed and denied entries are resolved against the working
// directory, like the paths they are compared with.
func TestSecurityRegression_RelativeEntriesFromWorkingDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "data"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	cfg := DefaultSecurityConfig()
	cfg.AllowedPaths = []string{"data"}
	for _, p := range []string{"data/x.json", filepath.Join(dir, "data", "x.json")} {
		if r := ValidateSecurePath(p, cfg); !r.IsValid {
			t.Errorf("%q rejected with AllowedPaths=[data]: %v", p, r.Errors)
		}
	}
	if r := ValidateSecurePath("other/x.json", cfg); r.IsValid {
		t.Errorf("other/x.json accepted with AllowedPaths=[data]")
	}

	cfg = DefaultSecurityConfig()
	cfg.DeniedPaths = []string{"./secret"}
	if r := ValidateSecurePath(filepath.Join(dir, "secret", "k"), cfg); r.IsValid {
		t.Errorf("absolute path into a relative denied entry accepted")
	}
}

// The original spelling stays in the result; only the comparison is absolute.
func TestSecurityRegression_RelativePathKeepsSpelling(t *testing.T) {
	r := ValidateSecurePath("data/./x.json", DefaultSecurityConfig())
	if r.Path != "data/./x.json" || r.NormalizedPath != filepath.Clean("data/x.json") {
		t.Errorf("Path=%q NormalizedPath=%q", r.Path, r.NormalizedPath)
	}
}

// A dangling link whose target has a missing directory after ".." is still
// resolved against the real location of the link's directory.
func TestSecurityRegression_DanglingLinkMissingDirAfterDotDot(t *testing.T) {
	work := t.TempDir()
	allowed := filepath.Join(work, "allowed")
	sub := filepath.Join(work, "outside", "sub")
	for _, d := range []string{filepath.Join(allowed, "nested"), sub} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(sub, filepath.Join(allowed, "nested", "jump")); err != nil {
		t.Skip(err)
	}
	if err := os.Symlink("../missing_dir/evil", filepath.Join(sub, "d")); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultSecurityConfig()
	cfg.AllowedPaths = []string{allowed}
	if r := ValidateSecurePath(filepath.Join(allowed, "nested", "jump", "d"), cfg); r.IsValid {
		t.Errorf("dangling link that targets %s accepted", filepath.Join(work, "outside", "missing_dir", "evil"))
	}
}

// aliasedDir returns real and alias, where alias is a symlink to real, like
// /var -> /private/var on macOS.
func aliasedDir(t *testing.T) (real, alias string) {
	t.Helper()
	root := t.TempDir()
	real = filepath.Join(root, "real")
	alias = filepath.Join(root, "alias")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, alias); err != nil {
		t.Skip(err)
	}
	return real, alias
}

// An allowed entry spelled through a link admits the files it leads to,
// whichever spelling the path and the working directory use.
func TestSecurityRegression_AllowedEntryThroughAlias(t *testing.T) {
	real, alias := aliasedDir(t)
	if err := os.Mkdir(filepath.Join(real, "data"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(alias)

	cfg := DefaultSecurityConfig()
	cfg.AllowedPaths = []string{alias}
	for _, p := range []string{"f.txt", filepath.Join(real, "f.txt"), filepath.Join(alias, "f.txt")} {
		if r := ValidateSecurePath(p, cfg); !r.IsValid {
			t.Errorf("%q rejected with AllowedPaths=[%s]: %v", p, alias, r.Errors)
		}
	}

	cfg.AllowedPaths = []string{"data"}
	for _, p := range []string{"data/x.json", filepath.Join(alias, "data", "x.json"), filepath.Join(real, "data", "x.json")} {
		if r := ValidateSecurePath(p, cfg); !r.IsValid {
			t.Errorf("%q rejected with AllowedPaths=[data] from %s: %v", p, alias, r.Errors)
		}
	}
}

// A denied entry spelled through a link denies the files it leads to, under
// any spelling.
func TestSecurityRegression_DeniedEntryThroughAlias(t *testing.T) {
	real, alias := aliasedDir(t)
	if err := os.Mkdir(filepath.Join(real, "secret"), 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultSecurityConfig()
	cfg.DeniedPaths = append(cfg.DeniedPaths, filepath.Join(alias, "secret"))
	for _, p := range []string{filepath.Join(real, "secret", "k"), filepath.Join(alias, "secret", "k")} {
		if r := ValidateSecurePath(p, cfg); r.IsValid {
			t.Errorf("%q accepted although %s is denied", p, filepath.Join(alias, "secret"))
		}
	}
	t.Chdir(real)
	if r := ValidateSecurePath("secret/k", cfg); r.IsValid {
		t.Errorf(`"secret/k" from %s accepted`, real)
	}
}
