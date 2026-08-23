package cli

func runAuxiliaryCommand(cmd string, args []string, version string) (int, bool) {
	switch cmd {
	case "completion":
		return completionCommand(args), true
	case "docs-manifest":
		return docsManifestCommand(args, version), true
	case "source-update":
		return sourceUpdateCommand(args), true
	default:
		return 0, false
	}
}
