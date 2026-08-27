package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// deepLinkTopic is a parsed reasonix://topic/... deep link. Only non-secret
// target identity is carried: topicId, scope, and (for project topics) the
// workspace root. Credentials, web tokens, full session file paths, and
// conversation bodies never appear in a deep link.
type deepLinkTopic struct {
	TopicID       string
	Scope         string // "project" | "global"
	WorkspaceRoot string // non-empty only when Scope == "project"
}

const deepLinkScheme = "reasonix"

// deepLinkQueueDepth bounds the buffered reasonix:// URLs. A burst of stale
// links (or a repeat handoff) drops oldest-first rather than growing memory.
const deepLinkQueueDepth = 8

// deepLink event channels surface activation results to the frontend so it can
// toast failures; the successful activation itself is already visible through
// the normal tab/topic events the frontend subscribes to.
const (
	deepLinkActivatedChannel = "deep-link:activated"
	deepLinkErrorChannel     = "deep-link:error"
)

var (
	errDeepLinkBadScheme   = errors.New("deep link must use the reasonix scheme")
	errDeepLinkBadHost     = errors.New("deep link host must be topic")
	errDeepLinkBadID       = errors.New("deep link is missing a usable topic id")
	errDeepLinkBadScope    = errors.New("deep link scope must be project or global")
	errDeepLinkBadParam    = errors.New("deep link has an unknown parameter")
	errDeepLinkNoWorkspace = errors.New("project deep link is missing a workspace root")
)

// isDeepLinkArg reports whether arg is a reasonix:// URL passed on the command
// line by the OS protocol handler.
func isDeepLinkArg(arg string) bool {
	return strings.HasPrefix(arg, deepLinkScheme+"://")
}

// parseDeepLink validates a reasonix:// topic deep link against the documented
// shape:
//
//	reasonix://topic/<topicId>?scope=project&workspace=<URL-encoded-root>
//	reasonix://topic/<topicId>?scope=global
//
// topicId, scope, and workspaceRoot are decoded here. The workspace root is
// additionally boundary-checked at activation time so a stale or foreign path
// is rejected before any tab is touched.
func parseDeepLink(raw string) (deepLinkTopic, error) {
	var out deepLinkTopic
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out, errDeepLinkBadScheme
	}
	u, err := url.Parse(raw)
	if err != nil {
		// A string that does not even parse as a URL (e.g. a corrupt escape
		// sequence) is not a usable reasonix:// deep link at all.
		return out, fmt.Errorf("%w: %v", errDeepLinkBadScheme, err)
	}
	if !strings.EqualFold(u.Scheme, deepLinkScheme) {
		return out, errDeepLinkBadScheme
	}
	if !strings.EqualFold(u.Host, "topic") {
		return out, errDeepLinkBadHost
	}
	topicID, err := singleTopicPathSegment(u.EscapedPath())
	if err != nil {
		return out, err
	}
	out.TopicID = topicID

	q := u.Query()
	scope := q.Get("scope")
	switch scope {
	case "project":
		out.Scope = "project"
		workspace := q.Get("workspace")
		if strings.TrimSpace(workspace) == "" {
			return out, errDeepLinkNoWorkspace
		}
		out.WorkspaceRoot = workspace
	case "global":
		out.Scope = "global"
		if strings.TrimSpace(q.Get("workspace")) != "" {
			return out, errDeepLinkBadParam
		}
	default:
		return out, errDeepLinkBadScope
	}
	for key := range q {
		switch key {
		case "scope", "workspace":
		default:
			return out, errDeepLinkBadParam
		}
	}
	return out, nil
}

// singleTopicPathSegment extracts exactly one non-empty path segment and
// decodes it. A topic id containing a literal slash (encoded %2F) is rejected
// so the id cannot span segments or smuggle a path.
func singleTopicPathSegment(escapedPath string) (string, error) {
	p := strings.TrimPrefix(escapedPath, "/")
	if p == "" || strings.Contains(p, "/") {
		return "", errDeepLinkBadID
	}
	decoded, err := url.PathUnescape(p)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errDeepLinkBadID, err)
	}
	if strings.TrimSpace(decoded) == "" || strings.Contains(decoded, "/") || strings.ContainsRune(decoded, 0) {
		return "", errDeepLinkBadID
	}
	return decoded, nil
}

// validateDeepLinkTarget is the workspace boundary check run right before
// activation: the project workspace root must be an absolute path to a
// directory that still exists. Global topics need no workspace.
func (d deepLinkTopic) validateTarget() error {
	if d.Scope == "global" {
		return nil
	}
	root := filepath.Clean(d.WorkspaceRoot)
	if root == "." || !filepath.IsAbs(root) || strings.ContainsRune(root, 0) {
		return fmt.Errorf("deep link workspace root must be an absolute path")
	}
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("deep link workspace root: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("deep link workspace root is not a directory")
	}
	return nil
}
