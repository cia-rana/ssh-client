package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	port = flag.Int("p", 22, "local port")
)

func connect() (*ssh.Client, error) {
	// Input username
	var username string
	fmt.Print("username: ")
	fmt.Scan(&username)

	// Input password(the password is not shown on the monitor)
	fmt.Print("password: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return nil, err
	}

	// Connect to remort host
	hostport := fmt.Sprintf("%s:%d", flag.Arg(0), *port)
	config := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{ssh.Password(string(password))},
		Timeout:         5 * time.Second,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil },
	}

	return ssh.Dial("tcp", hostport, config)
}

func run() int {
	flag.Parse()

	// Whether flag is valid
	if flag.NArg() == 0 {
		flag.Usage()
		return 2
	}

	// Connect to remote host
	conn, err := connect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	defer conn.Close()

	// Create new session
	session, err := conn.NewSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return HandleSessionError(err)
	}
	defer stdinPipe.Close()

	// Set pseudo terminal
	if err := session.RequestPty("xterm", 80, 40, ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}); err != nil {
		return HandleSessionError(err)
	}

	if err := session.Shell(); err != nil {
		return HandleSessionError(err)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		line, _ := reader.ReadString('\n')

		if _, err = fmt.Fprint(stdinPipe, line); err != nil {
			break
		}
	}

	return 0
}

func HandleSessionError(err error) int {
	fmt.Fprintf(os.Stderr, "%v\n", err)
	if ee, ok := err.(*ssh.ExitError); ok {
		return ee.ExitStatus()
	}
	return 1
}

func main() {
	os.Exit(run())
}
