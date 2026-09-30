package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"strings"

	"github.com/Shopify/go-lua"
)

var cmd string

type Lush struct {
	tasks   map[string]bool
	running []string
}

func main() {
	lush := &Lush{
		tasks: make(map[string]bool),
	}

	l := lua.NewState()

	l.Register("run", runCmd)
	l.Register("task", lush.registerTask)
	l.Register("run_task", lush.runTask)

	lua.OpenLibraries(l)

	if err := lua.DoFile(l, "lush.lua"); err != nil {
		fmt.Fprintf(os.Stderr, "lush: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		fmt.Println("No arguments provided.")
		fmt.Println("Usage: lush <task>\n")
		lush.listTasks()
		os.Exit(2)
	}

	taskName := os.Args[1]

	if !lush.tasks[taskName] {
		fmt.Fprintf(os.Stderr, "lush: task %q not found.\n", taskName)
		lush.listTasks()
		os.Exit(1)
	}

	l.Field(lua.RegistryIndex, "lush_task_"+taskName)
	if err := l.ProtectedCall(0, 0, 0); err != nil {
		fmt.Fprintf(os.Stderr, "lush: task %q failed: %v\n", taskName, err)
		os.Exit(1)
	}
}

func (lush Lush) listTasks() {
	fmt.Println("Available tasks:")
	for t := range lush.tasks {
		fmt.Fprintf(os.Stderr, "  - %s\n", t)
	}
}

func runCmd(l *lua.State) int {
	n := l.Top()
	if n == 0 {
		lua.Errorf(l, "run expects at least one command argument")
		panic("unreachable")
	}

	var args []string
	for i := 1; i <= n; i++ {
		str, ok := l.ToString(i)
		if !ok {
			lua.Errorf(l, "expected string at argument %d", i)
			panic("unreachable")
		}
		args = append(args, str)
	}

	// Handle OS interrupt signals
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			lua.Errorf(l, "\ncommand interrupted by user")
		} else {
			lua.Errorf(l, "command failed: %v", err)
		}
	}

	return 0
}

func (lush *Lush) registerTask(l *lua.State) int {
	name, ok := l.ToString(1)
	if !ok {
		lua.Errorf(l, "task() expected a string as first argument.")
		panic("unreachable")
	}

	if !l.IsFunction(2) {
		lua.Errorf(l, "task() expected function as second argument.")
		panic("unreachable")
	}

	lush.tasks[name] = true

	l.PushValue(2)
	l.SetField(lua.RegistryIndex, "lush_task_"+name)

	return 0
}

func (lush *Lush) runTask(l *lua.State) int {
	name, ok := l.ToString(1)
	if !ok {
		lua.Errorf(l, "run_task() expected a string as first argument.")
		panic("unreachable")
	}

	if !lush.tasks[name] {
		lua.Errorf(l, "task \"%s\" not found", name)
		panic("unreachable")
	}

	for _, running := range lush.running {
		if running == name {
			chain := append(append([]string{}, lush.running...), name ) 
			lua.Errorf(l, "circular task dependency %s", strings.Join(chain, " -> "))
			panic("unreachable")
		}
	}

	lush.running = append(lush.running, name)
	defer func() { lush.running = lush.running[:len(lush.running)-1] }()

	l.Field(lua.RegistryIndex, "lush_task_"+name)
	l.Call(0, 0)

	return 0
}
