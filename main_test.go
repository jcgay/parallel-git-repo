package main

import "os"

func ExampleRunCommand() {
	os.Args = []string{"parallel-git-repo"}

	main()

	// Output:
	// 1
}
