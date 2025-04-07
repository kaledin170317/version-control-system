


package main

import (
	"flag"
	"fmt"
	"os"
	"main/server"
	"os/signal"
	"context"

	"syscall"
)

func save(ctx context.Context, vcs *server.VCS) {
	commitHash1 := vcs.FixDirState(ctx)
	fmt.Printf("saved to commit hash: %s\n", commitHash1)
}

func all(ctx context.Context, vcs *server.VCS) {
	hahes := vcs.GetHistory(ctx)
	fmt.Printf("All hehes: %s\n", hahes)
	
}

func restoremy(ctx context.Context, vcs *server.VCS, hash string) {
	fmt.Printf(" restore to :) hash: %s\n", hash)
	vcs.GetDirState(ctx, hash)

}

func main() {

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	vcs := server.VCS{}
	dir := "/mnt/c/Users/kudimik/Desktop/web3/version-control-system/test"
	vcs.CreateVCS(ctx, dir)




	saveCmd := flag.NewFlagSet("save", flag.ExitOnError)

	restoreCmd := flag.NewFlagSet("restore", flag.ExitOnError)
	restoreHash := restoreCmd.String("hash", "/mnt/tmp/", "Include hash")

	allCmd := flag.NewFlagSet("all", flag.ExitOnError)

	if len(os.Args) < 2 {
		fmt.Println("Expected 'save', 'restore', or 'all' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "save":
		saveCmd.Parse(os.Args[2:])
		save(ctx, &vcs)
	case "restore":
		restoreCmd.Parse(os.Args[2:])
		if *restoreHash == "" {
			fmt.Println("Usage: restore --key <key> [--hash]")
			restoreCmd.PrintDefaults()
			os.Exit(1)
		}
		restoremy(ctx, &vcs, *restoreHash)
	case "all":
		allCmd.Parse(os.Args[2:])
		all(ctx, &vcs)
	default:
		fmt.Println("Unknown command. Available commands: save, restore, all")
		os.Exit(1)
	}
}
