package files

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/FrameworkOSS/portal/features/commands/handler"
	"github.com/FrameworkOSS/portal/portal"
	"github.com/fatih/color"
)

var (
	cyan   = color.New(color.FgCyan).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()

	aliasArgDir = []string{"d", "directory", "f", "folder"}
)

var (
	cmds     = []*handler.Command{cmdDirCh, cmdDirLs}
	cmdDirCh = handler.NewCommand().
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
				SetAliases(aliasArgDir...).
				SetType(handler.CommandArgTypeString).
				SetRequired(true).
				SetRequiresValue(true).
				SetRepeatable(true),
		)
	cmdDirLs = handler.NewCommand().
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
				SetAliases(aliasArgDir...).
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

type Files struct {
	lockResp  sync.Mutex
	processor *handler.EventCommandHandler
	resps     []*portal.Event

	workdir string
}

func NewFiles() (f *Files) {
	f = new(Files)
	f.resps = make([]*portal.Event, 0)
	f.processor = handler.NewEventCommandHandler()

	f.processor.GetEventHandler().
		Handle(f.eWorkdir, "workdir")

	f.processor.GetCommandHandler().
		Handle(f.cmdDirCh, cmdDirCh).
		Handle(f.cmdDirLs, cmdDirLs)

	return
}

func (f *Files) respond(ctx, r *portal.Event) {
	portal.EventClaim(ctx, r)
	go f.storeResp(r)
}

func (f *Files) eWorkdir(e *portal.Event) error {
	if e.GetDataSize() == 0 {
		f.storeResp(portal.NewEvent().SetID("workdir").SetData([]byte(f.workdir)).AddParticipants(e.GetProducer()))
		return nil
	}

	wd := string(e.GetData())
	if err := testDir(wd); err != nil {
		f.storeResp(portal.NewEventError(f.ID(), err).AddParticipants(e.GetProducer()))
		return nil
	}

	f.workdir = wd
	return nil
}

func (f *Files) setWorkdir(workdir string) {
	f.workdir = workdir
	f.respond(nil, portal.NewEvent().SetID("workdir").SetData([]byte(workdir)))
}

func (f *Files) cmdDirCh(cmd *handler.Command, e *portal.Event) error {
	wd := f.workdir
	dir := cmd.GetArgument("dir").GetValueString() //TODO: Switch to cmd.GetArguments("dir")

	if err := testDir(dir); err != nil {
		f.workdir = wd //Useless until handling multiple dir arguments!
		f.respond(e, portal.NewEventError(f.ID(), err))
		return fmt.Errorf("files: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		f.workdir = wd
		f.respond(e, portal.NewEventError(f.ID(), err))
		return fmt.Errorf("files: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		f.workdir = wd
		f.respond(e, portal.NewEventError(f.ID(), err))
		return fmt.Errorf("files: %v", err)
	}

	f.setWorkdir(wd)
	f.respond(e, portal.NewEventResponse(f.ID(), nil).AddParticipants(e.GetProducer()))
	return nil
}

func (f *Files) cmdDirLs(cmd *handler.Command, e *portal.Event) error {
	dir := f.workdir
	if test := cmd.GetArgument("dir"); test != nil { //TODO: Switch to cmd.GetArgumentsID("dir")
		dir = test.GetValueString()
	}
	nocolor := cmd.GetArgument("nocolor") != nil //True if specified

	if err := testDir(dir); err != nil {
		f.respond(e, portal.NewEventError(f.ID(), err))
		return fmt.Errorf("files: %v", err)
	}

	paths, err := os.ReadDir(dir)
	if err != nil {
		f.respond(e, portal.NewEventError(f.ID(), err))
		return fmt.Errorf("files: %v", err)
	}

	resp := ""
	for i := range paths {
		p := paths[i]
		perm := p.Type().Perm()
		name := p.Name()
		if p.IsDir() {
			name += "/"
		}

		if nocolor {
			resp += fmt.Sprintf("%s: %s\n", perm, name)
		} else {
			resp += fmt.Sprintf("%s: %s\n", yellow(perm), cyan(name))
		}
	}

	f.respond(e, portal.NewEventResponse(f.ID(), []byte(resp)).AddParticipants(e.GetProducer()))
	return nil
}

func testDir(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	stat, err := os.Stat(abs)
	if err != nil {
		return err
	}

	if !stat.IsDir() {
		return fmt.Errorf("files: %s is not a directory", path)
	}

	return nil
}

func (f *Files) storeResp(e *portal.Event) {
	f.lockResp.Lock()
	f.resps = append(f.resps, e)
	f.lockResp.Unlock()
}

func (f *Files) readResp() (e *portal.Event) {
	if len(f.resps) > 0 {
		f.lockResp.Lock()
		e = f.resps[0]
		f.resps = f.resps[1:]
		f.lockResp.Unlock()
	}
	return
}

func (f *Files) API() int {
	return 0
}

func (f *Files) ID() string {
	return "files"
}

func (f *Files) Name() string {
	return "Files"
}

func (f *Files) Authors() []string {
	return []string{"JoshuaDoes"}
}

func (f *Files) Description() string {
	return "Provides the capabilities of a standard file manager."
}

func (f *Files) Version() string {
	return "v0.0.1"
}

func (f *Files) Open() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	f.respond(nil, handler.NewEventCommandAdd(f.ID(), cmds...))
	f.respond(nil, portal.NewEventReady(f.ID(), true))
	f.setWorkdir(wd)
	return nil
}

func (f *Files) Close() (errs []error, retry bool) {
	return
}

func (f *Files) Input(e *portal.Event) error {
	return f.processor.Process(e)
}

func (f *Files) Output() (*portal.Event, error) {
	return f.readResp(), nil
}
