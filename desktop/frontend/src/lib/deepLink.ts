// Deep links are shareable reasonix:// topic URLs. They carry only non-secret
// target identity — the stable topicId plus, for project topics, the workspace
// root. No credentials, web tokens, session file paths, or conversation bodies
// ever enter a deep link.
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

// copyTopicDeepLink builds and copies the link, reporting success so the caller
// can toast feedback. Returns false when the session has no stable topicId (or
// a project topic without a workspace root) or the clipboard write failed.
export async function copyTopicDeepLink(session: SessionMeta): Promise<boolean> {
  const link = buildTopicDeepLink(session);
  if (!link) {
    return false;
  }
  return writeClipboardText(link);
}
