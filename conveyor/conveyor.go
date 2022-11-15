package conveyor

import (
	"sync"
)

func RunConveyor(tasksFlow ...task) {

	channels := []chan interface{}{}

	for i := 0; i < len(tasksFlow); i++ {
		ch := make(chan interface{})
		channels = append(channels, ch)
	}

	var wg sync.WaitGroup
	wg.Add(len(tasksFlow))

	for i, task := range tasksFlow {
		in := channels[i]
		out := channels[i]

		if i == len(channels)-1 {
			out = channels[0]
		} else {
			out = channels[i+1]
		}

		go func(task func(in, out chan interface{})) {
			task(in, out)
			close(out)
			wg.Done()

		}(task)

	}

	wg.Wait()

}
