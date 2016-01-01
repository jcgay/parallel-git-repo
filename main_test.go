package main

import "os"

func ExampleRunCommandForMultipleRepositories() {
	os.Args = []string{"parallel-git-repo", "echo"}

	main()

	// Output:
	// maven-notifier: /Users/jcgay/dev/maven-notifier
	// maven-color: /Users/jcgay/dev/maven-color
}
