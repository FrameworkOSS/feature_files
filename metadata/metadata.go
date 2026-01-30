package metadata

import "github.com/FrameworkOSS/feature_commands/handler"

const (
	API         = 0
	ID          = "files"
	Name        = "Files"
	Authors     = "JoshuaDoes"
	Description = "Provides the capabilities of a standard file manager."
	Version     = "v0.0.1"
)

var (
	AliasArgDir = []string{"d", "directory", "f", "folder"}

	Commands = []*handler.Command{CmdDirCh, CmdDirLs}
	CmdDirCh = handler.NewCommand().
			SetID("cd").
			SetName("Directory Change").
			SetAbout("Changes the current work directory on the filesystem.").
			SetUsage("Provide the relative or absolute path to change the work directory to.").
			SetAliases("chdir", "wd", "chwd").
			SetRequiresArguments(true).
			SetRequiresPreprocessing(true).
			SetArgument(handler.NewCommandArg().
				SetID("dir").
				SetName("directory").
				SetAbout("The directory to use.").
				SetUsage("Provide an absolute or relative path.").
				SetAliases(AliasArgDir...).
				SetType(handler.CommandArgTypeString).
				SetRequired(true).
				SetRequiresValue(true).
				SetRepeatable(true),
		)
	CmdDirLs = handler.NewCommand().
			SetID("ls").
			SetName("Directory List").
			SetAbout("Lists the contents of the current work directory.").
			SetUsage("Provide the relative or absolute path to list.").
			SetAliases("l", "dir", "lsdir", "ldir").
			SetRequiresArguments(true).
			SetRequiresPreprocessing(true).
			SetArgument(handler.NewCommandArg().
				SetID("dir").
				SetName("directory").
				SetAbout("The directory to use.").
				SetUsage("Provide an absolute or relative path.").
				SetAliases(AliasArgDir...).
				SetType(handler.CommandArgTypeString).
				SetRequiresValue(true).
				SetRepeatable(true),
		).SetArgument(handler.NewCommandArg().
		SetID("nocolor").
		SetName("No Color").
		SetAbout("Disables color rendering for terminal outputs.").
		SetAliases().
		SetType(handler.CommandArgTypeAuto),
	)
)
