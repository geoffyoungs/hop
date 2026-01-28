package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRsyncArgs_SingleFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantArchive bool
		wantVerbose bool
		wantCompress bool
		wantRecursive bool
		wantDryRun  bool
		wantDelete  bool
		wantSrc     string
		wantDst     string
	}{
		{
			name:        "archive flag",
			args:        []string{"-a", "src/", "server:/path"},
			wantArchive: true,
			wantSrc:     "src/",
			wantDst:     "server:/path",
		},
		{
			name:        "verbose flag",
			args:        []string{"-v", "src/", "server:/path"},
			wantVerbose: true,
			wantSrc:     "src/",
			wantDst:     "server:/path",
		},
		{
			name:         "compress flag",
			args:         []string{"-z", "src/", "server:/path"},
			wantCompress: true,
			wantSrc:      "src/",
			wantDst:      "server:/path",
		},
		{
			name:          "recursive flag",
			args:          []string{"-r", "src/", "server:/path"},
			wantRecursive: true,
			wantSrc:       "src/",
			wantDst:       "server:/path",
		},
		{
			name:       "dry-run short flag",
			args:       []string{"-n", "src/", "server:/path"},
			wantDryRun: true,
			wantSrc:    "src/",
			wantDst:    "server:/path",
		},
		{
			name:       "delete flag",
			args:       []string{"--delete", "src/", "server:/path"},
			wantDelete: true,
			wantSrc:    "src/",
			wantDst:    "server:/path",
		},
		{
			name:        "long archive flag",
			args:        []string{"--archive", "src/", "server:/path"},
			wantArchive: true,
			wantSrc:     "src/",
			wantDst:     "server:/path",
		},
		{
			name:        "long verbose flag",
			args:        []string{"--verbose", "src/", "server:/path"},
			wantVerbose: true,
			wantSrc:     "src/",
			wantDst:     "server:/path",
		},
		{
			name:         "long compress flag",
			args:         []string{"--compress", "src/", "server:/path"},
			wantCompress: true,
			wantSrc:      "src/",
			wantDst:      "server:/path",
		},
		{
			name:          "long recursive flag",
			args:          []string{"--recursive", "src/", "server:/path"},
			wantRecursive: true,
			wantSrc:       "src/",
			wantDst:       "server:/path",
		},
		{
			name:       "long dry-run flag",
			args:       []string{"--dry-run", "src/", "server:/path"},
			wantDryRun: true,
			wantSrc:    "src/",
			wantDst:    "server:/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, src, dst, err := parseRsyncArgs(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.wantArchive, opts.Archive)
			assert.Equal(t, tt.wantVerbose, opts.Verbose)
			assert.Equal(t, tt.wantCompress, opts.Compress)
			assert.Equal(t, tt.wantRecursive, opts.Recursive)
			assert.Equal(t, tt.wantDryRun, opts.DryRun)
			assert.Equal(t, tt.wantDelete, opts.Delete)
			assert.Equal(t, tt.wantSrc, src)
			assert.Equal(t, tt.wantDst, dst)
		})
	}
}

func TestParseRsyncArgs_CombinedFlags(t *testing.T) {
	opts, src, dst, err := parseRsyncArgs([]string{"-avz", "src/", "server:/path"})
	require.NoError(t, err)

	assert.True(t, opts.Archive)
	assert.True(t, opts.Verbose)
	assert.True(t, opts.Compress)
	assert.False(t, opts.Recursive)
	assert.False(t, opts.DryRun)
	assert.False(t, opts.Delete)
	assert.Equal(t, "src/", src)
	assert.Equal(t, "server:/path", dst)
}

func TestParseRsyncArgs_CombinedFlagsAVZR(t *testing.T) {
	opts, _, _, err := parseRsyncArgs([]string{"-avzr", "src/", "server:/path"})
	require.NoError(t, err)

	assert.True(t, opts.Archive)
	assert.True(t, opts.Verbose)
	assert.True(t, opts.Compress)
	assert.True(t, opts.Recursive)
}

func TestParseRsyncArgs_ExcludeWithEquals(t *testing.T) {
	opts, src, dst, err := parseRsyncArgs([]string{"--exclude=*.log", "src/", "server:/path"})
	require.NoError(t, err)

	assert.Equal(t, []string{"*.log"}, opts.Exclude)
	assert.Equal(t, "src/", src)
	assert.Equal(t, "server:/path", dst)
}

func TestParseRsyncArgs_ExcludeWithSpace(t *testing.T) {
	opts, src, dst, err := parseRsyncArgs([]string{"--exclude", "*.log", "src/", "server:/path"})
	require.NoError(t, err)

	assert.Equal(t, []string{"*.log"}, opts.Exclude)
	assert.Equal(t, "src/", src)
	assert.Equal(t, "server:/path", dst)
}

func TestParseRsyncArgs_MultipleExcludes(t *testing.T) {
	opts, _, _, err := parseRsyncArgs([]string{"--exclude=*.log", "--exclude", "tmp/", "src/", "server:/path"})
	require.NoError(t, err)

	assert.Equal(t, []string{"*.log", "tmp/"}, opts.Exclude)
}

func TestParseRsyncArgs_UnknownFlagsGoToExtra(t *testing.T) {
	opts, src, dst, err := parseRsyncArgs([]string{"-avzP", "src/", "server:/path"})
	require.NoError(t, err)

	// -avzP has unknown 'P', so entire flag goes to Extra
	assert.Equal(t, []string{"-avzP"}, opts.Extra)
	// But the known flags are still parsed
	assert.True(t, opts.Archive)
	assert.True(t, opts.Verbose)
	assert.True(t, opts.Compress)
	assert.Equal(t, "src/", src)
	assert.Equal(t, "server:/path", dst)
}

func TestParseRsyncArgs_ExtraLongFlags(t *testing.T) {
	// Unknown long flags that don't start with -- are positional
	// Unknown --flags should go to extra
	// But our parser only recognizes specific long flags
	// So unknown short flags with unrecognized chars go to Extra
	opts, _, _, err := parseRsyncArgs([]string{"-e", "src/", "server:/path"})
	require.NoError(t, err)

	assert.Equal(t, []string{"-e"}, opts.Extra)
}

func TestParseRsyncArgs_TooFewPositional(t *testing.T) {
	_, _, _, err := parseRsyncArgs([]string{"-avz", "src/"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exactly 2 positional arguments")
}

func TestParseRsyncArgs_TooManyPositional(t *testing.T) {
	_, _, _, err := parseRsyncArgs([]string{"src/", "server:/path", "extra"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exactly 2 positional arguments")
}

func TestParseRsyncArgs_NoArgs(t *testing.T) {
	_, _, _, err := parseRsyncArgs([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exactly 2 positional arguments")
}

func TestParseRsyncArgs_ExcludeMissingPattern(t *testing.T) {
	_, _, _, err := parseRsyncArgs([]string{"--exclude"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--exclude requires a pattern")
}

func TestParseRsyncArgs_DoubleDashSeparator(t *testing.T) {
	opts, src, dst, err := parseRsyncArgs([]string{"-a", "--", "-v", "server:/path"})
	require.NoError(t, err)

	assert.True(t, opts.Archive)
	// -v after -- is treated as positional, not a flag
	assert.False(t, opts.Verbose)
	assert.Equal(t, "-v", src)
	assert.Equal(t, "server:/path", dst)
}

func TestParseRsyncArgs_AllCombined(t *testing.T) {
	opts, src, dst, err := parseRsyncArgs([]string{
		"-avz", "--delete", "--exclude=*.log", "--exclude", "tmp/",
		"src/", "server:/path",
	})
	require.NoError(t, err)

	assert.True(t, opts.Archive)
	assert.True(t, opts.Verbose)
	assert.True(t, opts.Compress)
	assert.True(t, opts.Delete)
	assert.Equal(t, []string{"*.log", "tmp/"}, opts.Exclude)
	assert.Equal(t, "src/", src)
	assert.Equal(t, "server:/path", dst)
}
