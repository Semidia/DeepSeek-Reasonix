package cli

func upgradeCompletionSpec(help cliCompletionFlag) cliCompletionSpec {
	return completionSpecWithAliases("upgrade", []string{"update"}, []cliCompletionFlag{
		completionFlag("--check --force", cliCompletionNoValue), completionFlag("--channel", cliCompletionStaticValue), help,
	})
}

func sourceUpdateCompletionSpec(help cliCompletionFlag) cliCompletionSpec {
	return completionSpec("source-update", []cliCompletionFlag{
		completionFlag("--check --fetch --json", cliCompletionNoValue), completionFlag("--root", cliCompletionPathValue), help,
	})
}
