package bundle

import (
	"time"

	"github.com/databricks/bricks/libs/progress"
	"github.com/spf13/cobra"
)

// const AsciiEsc = "\033"

// var EraseLine = strings.Join([]string{AsciiEsc, "[2K"}, "")

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy bundle",

	RunE: func(cmd *cobra.Command, args []string) error {
		// fmt.Println("is terminal: ", term.IsTerminal(syscall.Stderr))
		// r := progress.NewDynamicRenderer(false)
		// l0 := r.AddEvent(progress.SpinnerStatePending, "line 0", 0)
		// // l1 := r.AddEvent(progress.SpinnerStatePending, "line 1", 0)
		// // l2 := r.AddEvent(progress.SpinnerStatePending, "line 2", 1)
		// // l3 := r.AddEvent(progress.SpinnerStatePending, "line 3", 2)
		// time.Sleep(time.Second)
		// r.UpdateContent(l0, "my line 0")
		// // fmt.Fprint(os.Stderr, "\033[H")
		// // r.UpdateContent(l0, "after returning home")
		// // r.UpdateContent(l1, "my line 1")
		// // r.UpdateContent(l2, "my line 2")
		// // r.UpdateContent(l3, "my line xxx")
		r := progress.NewEventRenderer()
		r.AddEvent(progress.EventStateRunning, "line 1", 0)
		r.AddEvent(progress.EventStateRunning, "line 2", 1)
		id := r.AddEvent(progress.EventStateRunning, "line 3", 2)
		r.AddEvent(progress.EventStateCompleted, "line 4", 0)
		r.AddEvent(progress.EventStateFailed, "line 5", 0)
		r.AddEvent(progress.EventStatePending, "line 6", 0)
		r.Start()
		time.Sleep(time.Second * 3)
		r.UpdateState(id, progress.EventStateCompleted)
		time.Sleep(time.Second * 3)
		r.UpdateContent(id, "done foo.")
		time.Sleep(time.Second * 3)
		r.Close()
		time.Sleep(time.Second * 5)
		// r.Render()
		// time.Sleep(time.Second)
		// r.Render()
		// time.Sleep(time.Second)
		// r.Render()
		// time.Sleep(time.Second)
		// r.Render()
		// fmt.Println("hello, world")
		return nil
	},
}

var force bool

func init() {
	AddCommand(deployCmd)
	deployCmd.Flags().BoolVar(&force, "force", false, "Force acquisition of deployment lock.")
}
