package main

import "os"

func ExampleRunCommandForMultipleRepositories() {
	os.Args = []string{"parallel-git-repo"}

	main()

	// Output:
	// /Users/jcgay/dev/maven-notifier
	// /Users/jcgay/dev/maven-color
}
