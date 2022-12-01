package conveyor

import (
	"sync"
)

func RunConveyor(tasksFlow ...task) {
	in := make(chan interface{})

	var wg sync.WaitGroup

	for _, task := range tasksFlow {
		out := make(chan interface{})

		wg.Add(1)
		go func(task func(in, out chan interface{}), in, out chan interface{}) {
			task(in, out)
			close(out)
			wg.Done()

		}(task, in, out)

		in = out
	}

	wg.Wait()

}
