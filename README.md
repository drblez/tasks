# Simple task manager

## Example


	dispatcher := NewDispatcher(2, 10, &SimpleLogger{})
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	dispatcher.Run(ctx, wg)
	f := func(i int) func() error {
		return func() error {
			fmt.Printf("Hi, I am function #%d\n", i)
			return nil
		}
	}
	i := 0
	for ; i < 20; i++ {
		dispatcher.Payload(f(i))
	}
	time.Sleep(2 * time.Second)
	cancel()
	wg.Wait()
