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
	blue   = color.New(color.FgHiBlue).SprintFunc()
	cyan   = color.New(color.FgHiCyan).SprintFunc()
	yellow = color.New(color.FgHiYellow).SprintFunc()
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

func (f *Files) SetWorkdir(wd string) error {
	if err := TestDir(wd); err != nil {
		return err
	}
	f.workdir = wd
	f.respond(nil, event.NewEvent().SetID("workdir").SetData([]byte(wd)))
	return nil
}
func (f *Files) GetWorkdir() string {
	return f.workdir
}
func (f *Files) eWorkdir(e *event.Event) error {
	if e.GetDataSize() == 0 {
		f.storeResp(event.NewEvent().SetID("workdir").SetData([]byte(f.workdir)).AddParticipants(e.GetProducer()))
		return nil
	}
	return f.SetWorkdir(string(e.GetData()))
}

func (f *Files) DirCh(dir ...string) error {
	og := f.workdir
	for i := range dir {
		d := dir[i]
		if err := TestDir(d); err != nil {
			f.workdir = og
			return err
		}

		if err := os.Chdir(d); err != nil {
			f.workdir = og
			return err
		}

		wd, err := os.Getwd()
		if err != nil {
			f.workdir = og
			return err
		}

		f.SetWorkdir(wd)
	}
	return nil
}
func (f *Files) cmdDirCh(cmd *handler.Command, e *event.Event) error {
	args := cmd.GetArgumentsID("dir")
	dirs := make([]string, len(args))
	for i := range args {
		dirs[i] = args[i].GetValueString()
	}

	if err := f.DirCh(dirs...); err != nil {
		f.respond(e, event.NewEventError(f.ID(), err))
		return fmt.Errorf("files: %v", err)
	}
	return nil
}

func (f *Files) DirLs(nocolor bool, dir ...string) (string, error) {
	if len(dir) == 0 {
		dir = []string{f.workdir}
	}

	resp := ""
	for i := range dir {
		d := dir[i]
		if err := TestDir(d); err != nil {
			return "", err
		}

		paths, err := os.ReadDir(d)
		if err != nil {
			return "", err
		}

		if i > 0 {
			resp += "\n"
		}

		for j := range paths {
			p := paths[j]
			name := p.Name()
			dest := ""
			perm := "??????????"

			if p.IsDir() {
				name += "/"
				perm = "d?????????"
			}
			if info, err := p.Info(); err == nil {
				perm = info.Mode().String()
				perm = strings.ToLower(string(perm[0])) + perm[1:]

				if perm[0] == 'l' {
					target, err := os.Readlink(d + string(os.PathSeparator) + name)
					if err == nil {
						dest = target
					}
				}
			}

			cPerm := perm
			cName := name
			if !nocolor {
				cPerm = yellow(perm)
				switch perm[0] {
				case 'l':
					cName = cyan(name)
				case 'd':
					cName = blue(name)
				}
			}
			if dest != "" {
				cName += " -> " + dest
			}
			resp += fmt.Sprintf("%s %s\n", cPerm, cName)
		}
	}
	return resp, nil
}
func (f *Files) cmdDirLs(cmd *handler.Command, e *event.Event) error {
	args := cmd.GetArgumentsID("dir")
	dirs := make([]string, len(args))
	for i := range args {
		dirs[i] = args[i].GetValueString()
	}
	nocolor := cmd.GetArgument("nocolor") != nil //True if specified

	resp, err := f.DirLs(nocolor, dirs...)
	if err != nil {
		f.respond(e, event.NewEventError(f.ID(), err))
		return fmt.Errorf("files: %v", err)
	}

	f.respond(e, event.NewEventResponse(f.ID(), []byte(resp)).AddParticipants(e.GetProducer()))
	return nil
}

func Stat(path string) (os.FileInfo, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return os.Stat(abs)
}
func TestDir(path string) error {
	stat, err := Stat(path)
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("files: %s is not a directory", path)
	}
	return nil
}
func TestFile(path string) error {
	stat, err := Stat(path)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return fmt.Errorf("files: %s is not a file", path)
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
	f.SetWorkdir(wd)
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
