package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/FrameworkOSS/event"
	"github.com/FrameworkOSS/feature_commands/handler"
	"github.com/FrameworkOSS/feature_files/metadata"
	"github.com/FrameworkOSS/portal"
	"github.com/fatih/color"
)

var (
	cyan   = color.New(color.FgCyan).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
)

type Files struct {
	lockResp  sync.Mutex
	processor *handler.EventCommandHandler
	resps     []*event.Event

	workdir string
}

func NewFiles() (f *Files) {
	f = new(Files)
	f.resps = make([]*event.Event, 0)
	f.processor = handler.NewEventCommandHandler()

	f.processor.GetEventHandler().
		Handle(f.eWorkdir, "workdir")

	f.processor.GetCommandHandler().
		Handle(f.cmdDirCh, metadata.CmdDirCh).
		Handle(f.cmdDirLs, metadata.CmdDirLs)

	return
}

func (f *Files) respond(ctx, r *event.Event) {
	portal.EventClaim(ctx, r)
	go f.storeResp(r)
}

func (f *Files) eWorkdir(e *event.Event) error {
	if e.GetDataSize() == 0 {
		f.storeResp(event.NewEvent().SetID("workdir").SetData([]byte(f.workdir)).AddParticipants(e.GetProducer()))
		return nil
	}

	wd := string(e.GetData())
	if err := testDir(wd); err != nil {
		f.storeResp(event.NewEventError(f.ID(), err).AddParticipants(e.GetProducer()))
		return nil
	}

	f.workdir = wd
	return nil
}

func (f *Files) setWorkdir(workdir string) {
	f.workdir = workdir
	f.respond(nil, event.NewEvent().SetID("workdir").SetData([]byte(workdir)))
}

func (f *Files) cmdDirCh(cmd *handler.Command, e *event.Event) error {
	wd := f.workdir
	dir := cmd.GetArgument("dir").GetValueString() //TODO: Switch to cmd.GetArguments("dir")

	if err := testDir(dir); err != nil {
		f.workdir = wd //Useless until handling multiple dir arguments!
		f.respond(e, event.NewEventError(f.ID(), err))
		return fmt.Errorf("files: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		f.workdir = wd
		f.respond(e, event.NewEventError(f.ID(), err))
		return fmt.Errorf("files: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		f.workdir = wd
		f.respond(e, event.NewEventError(f.ID(), err))
		return fmt.Errorf("files: %v", err)
	}

	f.setWorkdir(wd)
	f.respond(e, event.NewEventResponse(f.ID(), nil).AddParticipants(e.GetProducer()))
	return nil
}

func (f *Files) cmdDirLs(cmd *handler.Command, e *event.Event) error {
	dir := f.workdir
	if test := cmd.GetArgument("dir"); test != nil { //TODO: Switch to cmd.GetArgumentsID("dir")
		dir = test.GetValueString()
	}
	nocolor := cmd.GetArgument("nocolor") != nil //True if specified

	if err := testDir(dir); err != nil {
		f.respond(e, event.NewEventError(f.ID(), err))
		return fmt.Errorf("files: %v", err)
	}

	paths, err := os.ReadDir(dir)
	if err != nil {
		f.respond(e, event.NewEventError(f.ID(), err))
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

	f.respond(e, event.NewEventResponse(f.ID(), []byte(resp)).AddParticipants(e.GetProducer()))
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

func (f *Files) storeResp(e *event.Event) {
	f.lockResp.Lock()
	f.resps = append(f.resps, e)
	f.lockResp.Unlock()
}

func (f *Files) readResp() (e *event.Event) {
	if len(f.resps) > 0 {
		f.lockResp.Lock()
		e = f.resps[0]
		f.resps = f.resps[1:]
		f.lockResp.Unlock()
	}
	return
}

func (f *Files) API() int {
	return metadata.API
}

func (f *Files) ID() string {
	return metadata.ID
}

func (f *Files) Name() string {
	return metadata.Name
}

func (f *Files) Authors() []string {
	return strings.Split(metadata.Authors, ",")
}

func (f *Files) Description() string {
	return metadata.Description
}

func (f *Files) Version() string {
	return metadata.Version
}

func (f *Files) Open() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	f.respond(nil, handler.NewEventCommandAdd(f.ID(), metadata.Commands...))
	f.respond(nil, event.NewEventReady(f.ID(), true))
	f.setWorkdir(wd)
	return nil
}

func (f *Files) Close() (errs []error, retry bool) {
	return
}

func (f *Files) Input(e *event.Event) error {
	return f.processor.Process(e)
}

func (f *Files) Output() (*event.Event, error) {
	return f.readResp(), nil
}
