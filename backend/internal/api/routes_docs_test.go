package api

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestRESTDocsMentionRegisteredRoutes(t *testing.T) {
	routesSource, err := os.ReadFile("routes.go")
	if err != nil {
		t.Fatalf("read routes: %v", err)
	}
	docs, err := os.ReadFile("../../../docs/api/rest-api.md")
	if err != nil {
		t.Fatalf("read rest docs: %v", err)
	}
	docText := string(docs)
	routePattern := regexp.MustCompile(`HandleFunc\("([A-Z]+) ([^"]+)"`)
	matches := routePattern.FindAllStringSubmatch(string(routesSource), -1)
	if len(matches) == 0 {
		t.Fatalf("expected registered routes")
	}
	for _, match := range matches {
		method := match[1]
		path := match[2]
		if !strings.Contains(docText, path) {
			t.Fatalf("REST docs do not mention registered route %s %s", method, path)
		}
	}
}
