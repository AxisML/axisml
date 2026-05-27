package server

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLabelSelector_EmptyMatchesEverything(t *testing.T) {
	pred, err := ParseLabelSelector("")
	require.NoError(t, err)
	assert.True(t, pred(nil))
	assert.True(t, pred(map[string]string{"x": "y"}))
}

func TestParseLabelSelector_Equality(t *testing.T) {
	pred, err := ParseLabelSelector("a=b")
	require.NoError(t, err)
	assert.True(t, pred(map[string]string{"a": "b"}))
	assert.False(t, pred(map[string]string{"a": "c"}))
}

func TestParseLabelSelector_Inequality(t *testing.T) {
	pred, err := ParseLabelSelector("a!=b")
	require.NoError(t, err)
	assert.False(t, pred(map[string]string{"a": "b"}))
	assert.True(t, pred(map[string]string{"a": "c"}))
	assert.True(t, pred(map[string]string{}))
}

func TestParseLabelSelector_Existence(t *testing.T) {
	pred, err := ParseLabelSelector("a")
	require.NoError(t, err)
	assert.True(t, pred(map[string]string{"a": "anything"}))
	assert.False(t, pred(map[string]string{"b": "x"}))
}

func TestParseLabelSelector_NonExistence(t *testing.T) {
	pred, err := ParseLabelSelector("!a")
	require.NoError(t, err)
	assert.True(t, pred(map[string]string{}))
	assert.False(t, pred(map[string]string{"a": "x"}))
}

func TestParseLabelSelector_Invalid(t *testing.T) {
	_, err := ParseLabelSelector("!!bad")
	require.Error(t, err)
}

func TestJSONLabelsSQL_Empty(t *testing.T) {
	clause, args, err := JSONLabelsSQL("labels", "")
	require.NoError(t, err)
	assert.Empty(t, clause)
	assert.Empty(t, args)
}

func TestJSONLabelsSQL_Equality(t *testing.T) {
	clause, args, err := JSONLabelsSQL("labels", "a=b")
	require.NoError(t, err)
	assert.Equal(t, "(labels ->> ?) = ?", clause)
	assert.Equal(t, []any{"a", "b"}, args)
}

func TestJSONLabelsSQL_MultipleClausesANDed(t *testing.T) {
	clause, args, err := JSONLabelsSQL("labels", "a=1,!b")
	require.NoError(t, err)
	assert.Contains(t, clause, " AND ")
	assert.True(t, strings.Contains(clause, "(labels ->> ?) = ?"))
	// args contain key/value for the equality + key for !exists
	got := append([]any{}, args...)
	sort.Slice(got, func(i, j int) bool {
		return got[i].(string) < got[j].(string)
	})
	assert.Equal(t, []any{"1", "a", "b"}, got)
}
