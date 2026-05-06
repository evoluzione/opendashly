package builders

import (
	"strings"
	"testing"
)

func TestBuildLogsBodySearchClause_UsesLikeForUnderscoreToken(t *testing.T) {
	clause := buildLogsBodySearchClause("ZanolliShop_26000033")

	if strings.Contains(clause, "hasTokenCaseInsensitive(Body") {
		t.Fatalf("expected underscore token to avoid hasTokenCaseInsensitive: %s", clause)
	}
	if !strings.Contains(clause, "Body ILIKE '%ZanolliShop_26000033%'") {
		t.Fatalf("expected underscore token to use ILIKE: %s", clause)
	}
}

func TestBuildLogsBodySearchClause_UsesTokenSearchForAlnumToken(t *testing.T) {
	clause := buildLogsBodySearchClause("checkout123")

	if !strings.Contains(clause, "hasTokenCaseInsensitive(Body, 'checkout123')") {
		t.Fatalf("expected alnum token to use hasTokenCaseInsensitive: %s", clause)
	}
}
