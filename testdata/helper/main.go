package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	exitCode := flag.Int("exit", 0, "Exit code")
	sleepDuration := flag.Duration("sleep", 0, "Duration to sleep")
	stdoutMsg := flag.String("stdout", "", "Message to write to stdout")
	stderrMsg := flag.String("stderr", "", "Message to write to stderr")
	printPwd := flag.Bool("pwd", false, "Print working directory")
	printPid := flag.Bool("pid", false, "Print PID")
	trapGraceful := flag.Bool("graceful", false, "Trap SIGTERM/SIGINT and exit cleanly")
	ignoreSignals := flag.Bool("ignore-sig", false, "Ignore SIGTERM")
	spawnDepth := flag.Int("depth", 0, "Recursively spawn children to given depth")
	echoStdin := flag.Bool("stdin", false, "Echo stdin lines to stdout until EOF")

	flag.Parse()

	if *echoStdin {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			fmt.Printf("ECHO:%s\n", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "stdin read error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *printPid {
		fmt.Printf("PID=%d\n", os.Getpid())
	}

	if *printPwd {
		pwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "getwd error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(pwd)
	}

	if *stdoutMsg != "" {
		fmt.Println(*stdoutMsg)
	}

	if *stderrMsg != "" {
		fmt.Fprintln(os.Stderr, *stderrMsg)
	}

	if *spawnDepth > 0 {
		cmd := exec.Command(os.Args[0], fmt.Sprintf("-depth=%d", *spawnDepth-1), fmt.Sprintf("-sleep=%s", *sleepDuration))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "spawn error: %v\n", err)
			os.Exit(1)
		}
		defer cmd.Wait()
	}

	if *trapGraceful {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, os.Interrupt)
		<-sigCh
		// simulate clean shutdown delay
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}

	if *ignoreSignals {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, os.Interrupt)
		go func() {
			for range sigCh {
				// ignore
			}
		}()
	}

	if *sleepDuration > 0 {
		time.Sleep(*sleepDuration)
	}

	os.Exit(*exitCode)
}
