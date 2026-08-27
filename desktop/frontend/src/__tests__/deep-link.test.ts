import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { buildTopicDeepLink, copyTopicDeepLink } from "../lib/deepLink";

function session(overrides: Record<string, unknown> = {}) {
  return {
    path: "/sessions/x.json",
    preview: "hello",
    turns: 1,
    createdAt: 0,
    lastActivityAt: 0,
    modTime: 0,
    current: false,
    open: false,
    ...overrides,
  };
}

describe("buildTopicDeepLink", () => {
  it("builds a project link with an URL-encoded workspace root", () => {
    const link = buildTopicDeepLink(session({ topicId: "topic-42", scope: "project", workspaceRoot: "C:\\work\\demo" }));
    assert.equal(link, "reasonix://topic/topic-42?scope=project&workspace=C%3A%5Cwork%5Cdemo");
  });

  it("builds a global link with no workspace", () => {
    const link = buildTopicDeepLink(session({ topicId: "g-topic", scope: "global" }));
    assert.equal(link, "reasonix://topic/g-topic?scope=global");
  });

  it("falls back to global for a legacy session without scope", () => {
    const link = buildTopicDeepLink(session({ topicId: "legacy" }));
    assert.equal(link, "reasonix://topic/legacy?scope=global");
  });

  it("encodes a topic id with special characters", () => {
    const link = buildTopicDeepLink(session({ topicId: "a b+c#d", scope: "global" }));
    assert.equal(link, "reasonix://topic/a%20b%2Bc%23d?scope=global");
  });

  it("returns null without a topicId", () => {
    assert.equal(buildTopicDeepLink(session({})), null);
    assert.equal(buildTopicDeepLink(session({ topicId: "   " })), null);
  });

  it("returns null for a project topic without a workspace root", () => {
    assert.equal(buildTopicDeepLink(session({ topicId: "t", scope: "project" })), null);
    assert.equal(buildTopicDeepLink(session({ topicId: "t", scope: "project", workspaceRoot: "  " })), null);
  });
});

describe("copyTopicDeepLink", () => {
  it("reports failure for a session that cannot produce a link", async () => {
    assert.equal(await copyTopicDeepLink(session({})), false);
  });
});
