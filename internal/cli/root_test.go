package cli

import (
	"testing"

	"github.com/geoff/hop/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestParseAliasPrefix_Dot(t *testing.T) {
	alias, mode := parseAliasPrefix(".production")
	assert.Equal(t, "production", alias)
	assert.Equal(t, config.ModeProject, mode)
}

func TestParseAliasPrefix_Tilde(t *testing.T) {
	alias, mode := parseAliasPrefix("~production")
	assert.Equal(t, "production", alias)
	assert.Equal(t, config.ModeUser, mode)
}

func TestParseAliasPrefix_None(t *testing.T) {
	alias, mode := parseAliasPrefix("production")
	assert.Equal(t, "production", alias)
	assert.Equal(t, config.ModeDefault, mode)
}

func TestParseAliasPrefix_EmptyAfterDot(t *testing.T) {
	alias, mode := parseAliasPrefix(".")
	assert.Equal(t, "", alias)
	assert.Equal(t, config.ModeProject, mode)
}

func TestParseAliasPrefix_EmptyAfterTilde(t *testing.T) {
	alias, mode := parseAliasPrefix("~")
	assert.Equal(t, "", alias)
	assert.Equal(t, config.ModeUser, mode)
}

func TestParseAliasPrefix_DotInMiddle(t *testing.T) {
	// A dot in the middle should not be treated as a prefix
	alias, mode := parseAliasPrefix("production.staging")
	assert.Equal(t, "production.staging", alias)
	assert.Equal(t, config.ModeDefault, mode)
}

func TestParseAliasPrefix_TildeInMiddle(t *testing.T) {
	// A tilde in the middle should not be treated as a prefix
	alias, mode := parseAliasPrefix("production~staging")
	assert.Equal(t, "production~staging", alias)
	assert.Equal(t, config.ModeDefault, mode)
}
