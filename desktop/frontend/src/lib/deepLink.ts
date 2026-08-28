// Deep links are shareable reasonix:// URLs. Topic sessions carry a stable
// topicId; non-topic sessions have no id of their own, so their absolute
// session file path (URL-encoded) is used as the target identity instead. No
// credentials, web tokens, or conversation bodies ever enter a link.
import { writeClipboardText } from "./clipboard";
import type { SessionMeta } from "./sessionMetaTypes";

export function buildTopicDeepLink(session: SessionMeta): string | null {
  const topicId = session.topicId?.trim();
  if (!topicId) {
    return null;
  }
  const scope = session.scope === "project" ? "project" : "global";
  if (scope === "project") {
    const workspaceRoot = session.workspaceRoot?.trim();
    if (!workspaceRoot) {
      return null;
    }
    const params = new URLSearchParams({ scope, workspace: workspaceRoot });
    return `reasonix://topic/${encodeURIComponent(topicId)}?${params.toString()}`;
  }
  return `reasonix://topic/${encodeURIComponent(topicId)}?scope=global`;
}

// buildSessionDeepLink builds a deep link for any session: a topic link when
// the session has a topicId, otherwise a session link carrying its file path.
// Returns null only when the session has neither a topicId nor a usable path
// (or a project session without a workspace root).
export function buildSessionDeepLink(session: SessionMeta): string | null {
  const topicId = session.topicId?.trim();
  if (topicId) {
    return buildTopicDeepLink(session);
  }
  const path = session.path?.trim();
  if (!path) {
    return null;
  }
  const scope = session.scope === "project" ? "project" : "global";
  const params = new URLSearchParams({ scope });
  if (scope === "project") {
    const workspaceRoot = session.workspaceRoot?.trim();
    if (!workspaceRoot) {
      return null;
    }
    params.set("workspace", workspaceRoot);
  }
  return `reasonix://session/${encodeURIComponent(path)}?${params.toString()}`;
}

// copyTopicDeepLink builds and copies a topic deep link, reporting success so
// the caller can toast feedback. Returns false when the session has no stable
// topicId (or a project topic without a workspace root) or the clipboard write
// failed.
export async function copyTopicDeepLink(session: SessionMeta): Promise<boolean> {
  const link = buildTopicDeepLink(session);
  if (!link) {
    return false;
  }
  return writeClipboardText(link);
}

// copySessionDeepLink builds and copies a deep link for any session, reporting
// success so the caller can toast feedback.
export async function copySessionDeepLink(session: SessionMeta): Promise<boolean> {
  const link = buildSessionDeepLink(session);
  if (!link) {
    return false;
  }
  return writeClipboardText(link);
}
