package sqlsafe

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStatementsStripsComments(t *testing.T) {
	sql := `-- leading comment
SELECT 1; /* block */
WITH c AS (SELECT 1) SELECT * FROM c;`

	stmts, err := ParseStatements(sql)
	require.NoError(t, err)
	require.Len(t, stmts, 2)

	require.Equal(t, "SELECT 1", stmts[0].Text)
	require.Equal(t, "SELECT", stmts[0].FirstKeyword)
	require.Equal(t, Position{Line: 2, Column: 1}, stmts[0].FirstKeywordPos)

	require.Equal(t, "WITH c AS (SELECT 1) SELECT * FROM c", stmts[1].Text)
	require.Equal(t, "WITH", stmts[1].FirstKeyword)
	require.Equal(t, Position{Line: 3, Column: 1}, stmts[1].FirstKeywordPos)
}

func TestParseStatementsKeepsOriginalText(t *testing.T) {
	sql := "SELECT /*+ BROADCAST(dim) */ * FROM fact JOIN dim"

	stmts, err := ParseStatements(sql)
	require.NoError(t, err)
	require.Len(t, stmts, 1)
	require.Equal(t, sql, stmts[0].OriginalText)
}

func TestParseStatementsCollapsesWhitespaceFromComments(t *testing.T) {
	stmts, err := ParseStatements("SELECT/*hint*/ /*extra*/1")
	require.NoError(t, err)
	require.Len(t, stmts, 1)
	require.Equal(t, "SELECT 1", stmts[0].Text)
}

func TestClassifierAllowsReadOnlyStatements(t *testing.T) {
	stmts, err := ParseStatements("SELECT * FROM table")
	require.NoError(t, err)

	classifier := NewClassifier(DefaultPolicy())
	require.NoError(t, classifier.Check(stmts))
}

func TestClassifierBlocksDestructiveStatements(t *testing.T) {
	stmts, err := ParseStatements("INSERT INTO t VALUES (1)")
	require.NoError(t, err)

	classifier := NewClassifier(DefaultPolicy())
	err = classifier.Check(stmts)
	require.Error(t, err)

	violation, ok := err.(*Violation)
	require.True(t, ok)
	require.Equal(t, "INSERT", violation.Keyword)
	require.Equal(t, 0, violation.StatementIndex)
}

func TestClassifierWithSelectAllowed(t *testing.T) {
	stmts, err := ParseStatements("WITH c AS (SELECT 1) SELECT * FROM c")
	require.NoError(t, err)

	classifier := NewClassifier(DefaultPolicy())
	require.NoError(t, classifier.Check(stmts))
}

func TestClassifierWithInsertBlocked(t *testing.T) {
	stmts, err := ParseStatements("WITH c AS (SELECT 1) INSERT INTO t SELECT * FROM c")
	require.NoError(t, err)

	classifier := NewClassifier(DefaultPolicy())
	err = classifier.Check(stmts)
	require.Error(t, err)

	violation, ok := err.(*Violation)
	require.True(t, ok)
	require.Equal(t, "INSERT", violation.Keyword)
}

func TestClassifierExplainSelectAllowed(t *testing.T) {
	stmts, err := ParseStatements("EXPLAIN EXTENDED SELECT * FROM t")
	require.NoError(t, err)

	classifier := NewClassifier(DefaultPolicy())
	require.NoError(t, classifier.Check(stmts))
}

func TestClassifierExplainInsertBlocked(t *testing.T) {
	stmts, err := ParseStatements("EXPLAIN ANALYZE INSERT INTO t VALUES (1)")
	require.NoError(t, err)

	classifier := NewClassifier(DefaultPolicy())
	err = classifier.Check(stmts)
	require.Error(t, err)

	violation, ok := err.(*Violation)
	require.True(t, ok)
	require.Equal(t, "INSERT", violation.Keyword)
}

func TestTokenizeKeepsParentheses(t *testing.T) {
	tokens := tokenize("WITH c AS (SELECT 1) SELECT")
	require.Equal(t, []string{"WITH", "C", "AS", "(", "SELECT", "1", ")", "SELECT"}, tokens)
}

func TestClassifierIgnoresKeywordsInStringLiterals(t *testing.T) {
	stmts, err := ParseStatements("SELECT 'drop table users' AS note")
	require.NoError(t, err)

	classifier := NewClassifier(DefaultPolicy())
	require.NoError(t, classifier.Check(stmts))
}

func TestClassifierIgnoresKeywordsInQuotedIdentifiers(t *testing.T) {
	stmts, err := ParseStatements("SELECT \"drop\" AS col FROM data")
	require.NoError(t, err)

	classifier := NewClassifier(DefaultPolicy())
	require.NoError(t, classifier.Check(stmts))
}
